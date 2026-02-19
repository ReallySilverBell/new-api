package model

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// ChannelDynamicState 渠道动态状态
type ChannelDynamicState struct {
	ChannelId        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	DynamicFactor    float64 `json:"dynamic_factor"`    // 当前动态权重因子 [0.0039, 1.0]
	ConsecutiveFails int     `json:"consecutive_fails"` // 连续失败次数 [0, 5+]
	LastFailTime     int64   `json:"last_fail_time"`    // 最后失败时间戳
	LastSuccessTime  int64   `json:"last_success_time"` // 最后成功时间戳
	CooldownEndTime  int64   `json:"cooldown_end_time"` // 冷却结束时间戳
	TotalRequests    int64   `json:"total_requests"`    // 总请求数
	FailedRequests   int64   `json:"failed_requests"`   // 失败请求数
	SuccessRate      float64 `json:"success_rate"`      // 成功率
	EffectiveWeight  int     `json:"effective_weight"`  // 有效权重（静态×动态）
	StaticWeight     int     `json:"static_weight"`     // 静态权重
	Status           string  `json:"status"`            // 状态描述
	mu               sync.RWMutex
}

// DynamicWeightConfig 动态权重配置
type DynamicWeightConfig struct {
	Enabled             bool    `json:"enabled"`               // 是否启用，默认 true
	RecoveryStep        float64 `json:"recovery_step"`         // 成功恢复步长，默认 0.2 (20%)
	CooldownPeriod      int64   `json:"cooldown_period"`       // 冷却期（秒），默认 3600 (1小时)
	TimeRecoveryRate    float64 `json:"time_recovery_rate"`    // 时间恢复速率，默认 0.1 (10%/小时)
	MinFactor           float64 `json:"min_factor"`            // 最小因子，默认 0.00390625
	MaxConsecutiveFails int     `json:"max_consecutive_fails"` // 最大连续失败，默认 4
}

// 全局变量
var (
	channelDynamicStates = make(map[int]*ChannelDynamicState)
	dynamicStateLock     sync.RWMutex
	dynamicWeightConfig  = DynamicWeightConfig{
		Enabled:             true,
		RecoveryStep:        0.2,
		CooldownPeriod:      3600,
		TimeRecoveryRate:    0.1,
		MinFactor:           0.00390625,
		MaxConsecutiveFails: 4,
	}
	recoveryTicker *time.Ticker
)

// InitDynamicWeight 初始化动态权重系统
func InitDynamicWeight() {
	if !dynamicWeightConfig.Enabled {
		common.SysLog("Dynamic weight system is disabled")
		return
	}

	// 启动后台恢复任务
	if recoveryTicker != nil {
		recoveryTicker.Stop()
	}
	recoveryTicker = time.NewTicker(1 * time.Hour)
	go func() {
		for range recoveryTicker.C {
			RecoverChannelsByTime()
		}
	}()

	common.SysLog("Dynamic weight system initialized")
}

// GetOrCreateDynamicState 获取或创建动态状态
func GetOrCreateDynamicState(channelId int, channelName string, staticWeight int) *ChannelDynamicState {
	dynamicStateLock.Lock()
	defer dynamicStateLock.Unlock()

	if state, exists := channelDynamicStates[channelId]; exists {
		return state
	}

	state := &ChannelDynamicState{
		ChannelId:        channelId,
		ChannelName:      channelName,
		DynamicFactor:    1.0,
		ConsecutiveFails: 0,
		StaticWeight:     staticWeight,
		LastSuccessTime:  time.Now().Unix(),
	}
	channelDynamicStates[channelId] = state
	return state
}

// GetDynamicState 获取动态状态（只读）
func GetDynamicState(channelId int) *ChannelDynamicState {
	dynamicStateLock.RLock()
	defer dynamicStateLock.RUnlock()
	return channelDynamicStates[channelId]
}

// OnChannelFailure 处理渠道失败
func OnChannelFailure(channelId int, channelName string, staticWeight int, shouldPenalize bool) {
	if !dynamicWeightConfig.Enabled {
		return
	}

	state := GetOrCreateDynamicState(channelId, channelName, staticWeight)
	state.mu.Lock()
	defer state.mu.Unlock()

	state.TotalRequests++
	state.FailedRequests++

	// 只对应该惩罚的错误降低权重
	if !shouldPenalize {
		state.updateSuccessRate()
		return
	}

	state.ConsecutiveFails++

	// 计算指数：1次→1, 2次→2, 3次→4, 4次及以上→8
	var exponent int
	switch state.ConsecutiveFails {
	case 1:
		exponent = 1
	case 2:
		exponent = 2
	case 3:
		exponent = 4
	default:
		exponent = 8 // 保持最低值
	}

	oldFactor := state.DynamicFactor
	state.DynamicFactor = math.Pow(0.5, float64(exponent))
	state.LastFailTime = time.Now().Unix()
	state.CooldownEndTime = state.LastFailTime + dynamicWeightConfig.CooldownPeriod

	state.updateSuccessRate()

	common.SysLog(fmt.Sprintf("Channel #%d (%s) failed %d times, dynamic factor: %.4f -> %.4f",
		channelId, channelName, state.ConsecutiveFails, oldFactor, state.DynamicFactor))
}

// OnChannelSuccess 处理渠道成功
func OnChannelSuccess(channelId int, channelName string, staticWeight int) {
	if !dynamicWeightConfig.Enabled {
		return
	}

	state := GetOrCreateDynamicState(channelId, channelName, staticWeight)
	state.mu.Lock()
	defer state.mu.Unlock()

	state.TotalRequests++

	// 使用更激进的快速复原算法: (当前因子 + 5%) * 2
	oldFactor := state.DynamicFactor
	state.DynamicFactor = math.Min(1.0, (state.DynamicFactor+0.05)*2.0)

	// 如果完全恢复，重置连续失败计数
	if state.DynamicFactor >= 1.0 {
		state.ConsecutiveFails = 0
	}

	state.LastSuccessTime = time.Now().Unix()
	state.updateSuccessRate()

	if oldFactor < 1.0 {
		common.SysLog(fmt.Sprintf("Channel #%d (%s) success, dynamic factor: %.4f -> %.4f",
			channelId, channelName, oldFactor, state.DynamicFactor))
	}
}

// GetDynamicFactor 获取渠道的动态权重因子
func GetDynamicFactor(channelId int) float64 {
	if !dynamicWeightConfig.Enabled {
		return 1.0
	}

	state := GetDynamicState(channelId)
	if state == nil {
		return 1.0
	}

	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.DynamicFactor
}

// RecoverChannelsByTime 时间恢复任务（每小时执行）
func RecoverChannelsByTime() {
	if !dynamicWeightConfig.Enabled {
		return
	}

	dynamicStateLock.RLock()
	states := make([]*ChannelDynamicState, 0, len(channelDynamicStates))
	for _, state := range channelDynamicStates {
		states = append(states, state)
	}
	dynamicStateLock.RUnlock()

	currentTime := time.Now().Unix()
	recoveredCount := 0

	for _, state := range states {
		state.mu.Lock()

		// 检查是否还在冷却期
		if currentTime < state.CooldownEndTime {
			state.mu.Unlock()
			continue
		}

		// 检查是否需要恢复
		if state.DynamicFactor < 1.0 {
			// 计算已过多少小时（从冷却结束开始）
			hoursSinceCooldown := float64(currentTime-state.CooldownEndTime) / 3600.0

			// 每小时恢复 TimeRecoveryRate (默认10%)
			recoveryAmount := dynamicWeightConfig.TimeRecoveryRate * hoursSinceCooldown
			oldFactor := state.DynamicFactor
			state.DynamicFactor = math.Min(1.0, state.DynamicFactor+recoveryAmount)

			// 如果完全恢复，重置连续失败计数
			if state.DynamicFactor >= 1.0 {
				state.ConsecutiveFails = 0
			}

			recoveredCount++
			common.SysLog(fmt.Sprintf("Channel #%d (%s) time recovery: %.4f -> %.4f (%.1f hours since cooldown)",
				state.ChannelId, state.ChannelName, oldFactor, state.DynamicFactor, hoursSinceCooldown))
		}

		state.mu.Unlock()
	}

	if recoveredCount > 0 {
		common.SysLog(fmt.Sprintf("Time recovery completed: %d channels recovered", recoveredCount))
	}
}

// GetAllChannelDynamicStates 获取所有渠道的动态状态（用于API）
// 包含所有已启用的渠道，未产生请求的渠道显示默认值（100%权重）
func GetAllChannelDynamicStates() []ChannelDynamicState {
	dynamicStateLock.RLock()
	defer dynamicStateLock.RUnlock()

	currentTime := time.Now().Unix()

	// 已有动态状态的渠道ID集合
	hasDynamicState := make(map[int]bool)

	states := make([]ChannelDynamicState, 0)

	// 先添加有动态状态记录的渠道
	for _, state := range channelDynamicStates {
		state.mu.RLock()

		// 计算有效权重
		effectiveWeight := int(float64(state.StaticWeight) * state.DynamicFactor)

		// 计算状态描述
		status := getStatusDescription(state, currentTime)

		states = append(states, ChannelDynamicState{
			ChannelId:        state.ChannelId,
			ChannelName:      state.ChannelName,
			DynamicFactor:    state.DynamicFactor,
			ConsecutiveFails: state.ConsecutiveFails,
			LastFailTime:     state.LastFailTime,
			LastSuccessTime:  state.LastSuccessTime,
			CooldownEndTime:  state.CooldownEndTime,
			TotalRequests:    state.TotalRequests,
			FailedRequests:   state.FailedRequests,
			SuccessRate:      state.SuccessRate,
			EffectiveWeight:  effectiveWeight,
			StaticWeight:     state.StaticWeight,
			Status:           status,
		})

		hasDynamicState[state.ChannelId] = true
		state.mu.RUnlock()
	}

	// 获取所有已启用但尚未有动态记录的渠道
	if common.MemoryCacheEnabled {
		// 从内存缓存获取
		channelSyncLock.RLock()
		for id, channel := range channelsIDM {
			if hasDynamicState[id] {
				continue
			}
			if channel.Status != common.ChannelStatusEnabled {
				continue
			}
			staticWeight := channel.GetWeight()
			if staticWeight == 0 {
				staticWeight = 100
			}
			states = append(states, ChannelDynamicState{
				ChannelId:       id,
				ChannelName:     channel.Name,
				DynamicFactor:   1.0,
				SuccessRate:     1.0,
				EffectiveWeight: staticWeight,
				StaticWeight:    staticWeight,
				Status:          "正常",
			})
		}
		channelSyncLock.RUnlock()
	} else {
		// 内存缓存未启用，从数据库查询
		var channels []*Channel
		DB.Where("status = ?", common.ChannelStatusEnabled).Find(&channels)
		for _, channel := range channels {
			if hasDynamicState[channel.Id] {
				continue
			}
			staticWeight := channel.GetWeight()
			if staticWeight == 0 {
				staticWeight = 100
			}
			states = append(states, ChannelDynamicState{
				ChannelId:       channel.Id,
				ChannelName:     channel.Name,
				DynamicFactor:   1.0,
				SuccessRate:     1.0,
				EffectiveWeight: staticWeight,
				StaticWeight:    staticWeight,
				Status:          "正常",
			})
		}
	}

	// 按渠道ID排序
	sort.Slice(states, func(i, j int) bool {
		return states[i].ChannelId < states[j].ChannelId
	})

	return states
}

// GetDynamicWeightConfig 获取配置
func GetDynamicWeightConfig() DynamicWeightConfig {
	return dynamicWeightConfig
}

// SetDynamicWeightConfig 设置配置
func SetDynamicWeightConfig(config DynamicWeightConfig) {
	dynamicWeightConfig = config
	if config.Enabled {
		InitDynamicWeight()
	} else if recoveryTicker != nil {
		recoveryTicker.Stop()
		recoveryTicker = nil
	}
}

// 内部辅助方法

func (s *ChannelDynamicState) updateSuccessRate() {
	if s.TotalRequests > 0 {
		s.SuccessRate = float64(s.TotalRequests-s.FailedRequests) / float64(s.TotalRequests)
	} else {
		s.SuccessRate = 1.0
	}
}

func getStatusDescription(state *ChannelDynamicState, currentTime int64) string {
	if state.DynamicFactor >= 1.0 {
		return "正常"
	}

	if currentTime < state.CooldownEndTime {
		remainingMinutes := (state.CooldownEndTime - currentTime) / 60
		return fmt.Sprintf("冷却中 (剩余%d分钟)", remainingMinutes)
	}

	if state.DynamicFactor < 0.1 {
		return "严重降权"
	}

	if state.DynamicFactor < 0.5 {
		return "降权中"
	}

	return "恢复中"
}
