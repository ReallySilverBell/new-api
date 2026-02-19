package controller

import (
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type ChannelMetricsResponse struct {
	Enabled   bool                          `json:"enabled"`    // 动态权重是否启用
	Config    model.DynamicWeightConfig     `json:"config"`     // 当前配置
	Channels  []model.ChannelDynamicState   `json:"channels"`   // 所有渠道状态
	UpdatedAt int64                         `json:"updated_at"` // 更新时间
}

// GetChannelDynamicMetrics 获取所有渠道的动态权重状态
// GET /api/channel/dynamic-metrics
func GetChannelDynamicMetrics(c *gin.Context) {
	config := model.GetDynamicWeightConfig()
	channels := model.GetAllChannelDynamicStates()

	c.JSON(http.StatusOK, ChannelMetricsResponse{
		Enabled:   config.Enabled,
		Config:    config,
		Channels:  channels,
		UpdatedAt: time.Now().Unix(),
	})
}

// UpdateDynamicWeightConfig 更新动态权重配置
// PUT /api/channel/dynamic-config
func UpdateDynamicWeightConfig(c *gin.Context) {
	var config model.DynamicWeightConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid configuration: " + err.Error(),
		})
		return
	}

	// 验证配置
	if config.RecoveryStep <= 0 || config.RecoveryStep > 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "RecoveryStep must be between 0 and 1",
		})
		return
	}

	if config.TimeRecoveryRate <= 0 || config.TimeRecoveryRate > 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "TimeRecoveryRate must be between 0 and 1",
		})
		return
	}

	if config.CooldownPeriod < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "CooldownPeriod must be non-negative",
		})
		return
	}

	model.SetDynamicWeightConfig(config)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration updated successfully",
		"data":    config,
	})
}

// GetDynamicWeightConfig 获取动态权重配置
// GET /api/channel/dynamic-config
func GetDynamicWeightConfig(c *gin.Context) {
	config := model.GetDynamicWeightConfig()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// TriggerTimeRecovery 手动触发时间恢复（用于测试）
// POST /api/channel/trigger-recovery
func TriggerTimeRecovery(c *gin.Context) {
	model.RecoverChannelsByTime()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Time recovery triggered successfully",
	})
}
