/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Card,
  Table,
  Tag,
  Typography,
  Space,
  Button,
  Spin,
  Banner,
  Descriptions,
  Switch,
  InputNumber,
  Modal,
  Toast,
} from '@douyinfe/semi-ui';
import { API } from '../../helpers';
import { IconRefresh, IconSetting } from '@douyinfe/semi-icons';

const { Title, Text } = Typography;

const DynamicWeight = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState(null);
  const [configModalVisible, setConfigModalVisible] = useState(false);
  const [config, setConfig] = useState(null);
  const [saving, setSaving] = useState(false);

  // 加载数据
  const loadData = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/channel/dynamic-metrics');
      const { success, message, data } = res.data;
      if (success === false) {
        Toast.error(message || '加载失败');
        return;
      }
      setData(res.data);
    } catch (error) {
      Toast.error('加载失败: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  // 加载配置（后端存小数，前端显示百分比，加载时转换）
  const loadConfig = async () => {
    try {
      const res = await API.get('/api/channel/dynamic-config');
      const { success, message, data } = res.data;
      if (success === false) {
        Toast.error(message || '加载配置失败');
        return;
      }
      setConfig({
        ...data,
        time_recovery_rate: Math.round(data.time_recovery_rate * 100),
        min_factor: Math.round(data.min_factor * 10000) / 100,
      });
    } catch (error) {
      Toast.error('加载配置失败: ' + error.message);
    }
  };

  // 保存配置（前端存百分比整数，保存时转回小数给后端）
  const saveConfig = async () => {
    setSaving(true);
    try {
      const payload = {
        ...config,
        time_recovery_rate: config.time_recovery_rate / 100,
        min_factor: config.min_factor / 100,
      };
      const res = await API.put('/api/channel/dynamic-config', payload);
      const { success, message } = res.data;
      if (success === false) {
        Toast.error(message || '保存失败');
        return;
      }
      Toast.success('配置已更新');
      setConfigModalVisible(false);
      await loadData();
    } catch (error) {
      Toast.error('保存失败: ' + error.message);
    } finally {
      setSaving(false);
    }
  };

  // 手动触发恢复
  const triggerRecovery = async () => {
    try {
      const res = await API.post('/api/channel/trigger-recovery');
      const { success, message } = res.data;
      if (success === false) {
        Toast.error(message || '触发失败');
        return;
      }
      Toast.success('时间恢复已触发');
      await loadData();
    } catch (error) {
      Toast.error('触发失败: ' + error.message);
    }
  };

  useEffect(() => {
    loadData();
    loadConfig();
    // 每30秒自动刷新
    const interval = setInterval(loadData, 30000);
    return () => clearInterval(interval);
  }, []);

  // 获取状态标签
  const getStatusTag = (status, dynamicFactor) => {
    if (status === '正常') {
      return <Tag color='green'>{status}</Tag>;
    } else if (status.includes('冷却中')) {
      return <Tag color='blue'>{status}</Tag>;
    } else if (status === '严重降权') {
      return <Tag color='red'>{status}</Tag>;
    } else if (status === '降权中') {
      return <Tag color='orange'>{status}</Tag>;
    } else if (status === '恢复中') {
      return <Tag color='cyan'>{status}</Tag>;
    }
    return <Tag>{status}</Tag>;
  };

  // 格式化时间
  const formatTime = (timestamp) => {
    if (!timestamp || timestamp === 0) return '-';
    return new Date(timestamp * 1000).toLocaleString('zh-CN');
  };

  // 格式化百分比
  const formatPercent = (value) => {
    return (value * 100).toFixed(2) + '%';
  };

  // 表格列定义
  const columns = [
    {
      title: 'ID',
      dataIndex: 'channel_id',
      width: 80,
    },
    {
      title: '渠道名称',
      dataIndex: 'channel_name',
      width: 150,
    },
    {
      title: '动态因子',
      dataIndex: 'dynamic_factor',
      width: 120,
      render: (value) => (
        <Text strong style={{ color: value < 0.5 ? '#f5222d' : '#52c41a' }}>
          {formatPercent(value)}
        </Text>
      ),
    },
    {
      title: '有效权重',
      dataIndex: 'effective_weight',
      width: 100,
      render: (value, record) => (
        <Text>
          {value} <Text type='tertiary'>/ {record.static_weight}</Text>
        </Text>
      ),
    },
    {
      title: '连续失败',
      dataIndex: 'consecutive_fails',
      width: 100,
      render: (value) => (
        <Text strong style={{ color: value > 0 ? '#f5222d' : '#52c41a' }}>
          {value}
        </Text>
      ),
    },
    {
      title: '成功率',
      dataIndex: 'success_rate',
      width: 100,
      render: (value) => (
        <Text style={{ color: value < 0.9 ? '#f5222d' : '#52c41a' }}>
          {formatPercent(value)}
        </Text>
      ),
    },
    {
      title: '总请求',
      dataIndex: 'total_requests',
      width: 100,
    },
    {
      title: '失败请求',
      dataIndex: 'failed_requests',
      width: 100,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 150,
      render: (value, record) => getStatusTag(value, record.dynamic_factor),
    },
    {
      title: '最后成功',
      dataIndex: 'last_success_time',
      width: 180,
      render: (value) => <Text type='tertiary'>{formatTime(value)}</Text>,
    },
    {
      title: '最后失败',
      dataIndex: 'last_fail_time',
      width: 180,
      render: (value) => <Text type='tertiary'>{formatTime(value)}</Text>,
    },
  ];

  return (
    <div className='mt-[60px] px-2'>
      <div style={{ padding: '20px' }}>
        <Space vertical align='start' spacing='large' style={{ width: '100%' }}>
          {/* 标题和操作按钮 */}
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              width: '100%',
            }}
          >
            <Title heading={3}>动态权重管理</Title>
            <Space>
              <Button
                icon={<IconRefresh />}
                onClick={loadData}
                loading={loading}
              >
                刷新
              </Button>
              <Button
                icon={<IconSetting />}
                onClick={() => setConfigModalVisible(true)}
              >
                配置
              </Button>
              <Button onClick={triggerRecovery} type='tertiary'>
                触发时间恢复
              </Button>
            </Space>
          </div>

          {/* 系统状态横幅 */}
          {data && (
            <Banner
              type={data.enabled ? 'info' : 'warning'}
              description={
                data.enabled
                  ? '动态权重系统已启用，正在自动调整渠道权重'
                  : '动态权重系统已禁用'
              }
              closeIcon={null}
            />
          )}

          {/* 配置信息卡片 */}
          {data && data.config && (
            <Card title='当前配置' style={{ width: '100%' }}>
              <Descriptions
                data={[
                  {
                    key: '启用状态',
                    value: data.config.enabled ? (
                      <Tag color='green'>已启用</Tag>
                    ) : (
                      <Tag color='red'>已禁用</Tag>
                    ),
                  },
                  {
                    key: '冷却期',
                    value: `${data.config.cooldown_period / 60} 分钟`,
                  },
                  {
                    key: '时间恢复速率',
                    value: `${formatPercent(data.config.time_recovery_rate)}/小时`,
                  },
                  {
                    key: '最小因子',
                    value: formatPercent(data.config.min_factor),
                  },
                  {
                    key: '最大连续失败',
                    value: data.config.max_consecutive_fails,
                  },
                ]}
                row
              />
            </Card>
          )}

          {/* 渠道列表 */}
          <Card title='渠道动态权重状态' style={{ width: '100%' }}>
            <Spin spinning={loading}>
              <Table
                columns={columns}
                dataSource={data?.channels || []}
                pagination={{
                  pageSize: 20,
                  showSizeChanger: true,
                  pageSizeOpts: [10, 20, 50, 100],
                }}
                rowKey='channel_id'
                empty='暂无数据'
              />
            </Spin>
          </Card>

          {/* 更新时间 */}
          {data && (
            <Text type='tertiary' size='small'>
              最后更新: {formatTime(data.updated_at)}
            </Text>
          )}
        </Space>

        {/* 配置模态框 */}
        <Modal
          title='动态权重配置'
          visible={configModalVisible}
          onOk={saveConfig}
          onCancel={() => setConfigModalVisible(false)}
          confirmLoading={saving}
          width={600}
        >
          {config && (
            <Space vertical align='start' spacing='large' style={{ width: '100%' }}>
              <div>
                <Text strong>启用动态权重</Text>
                <br />
                <Switch
                  checked={config.enabled}
                  onChange={(checked) =>
                    setConfig({ ...config, enabled: checked })
                  }
                />
              </div>

              <div>
                <Text strong>冷却期 (秒)</Text>
                <br />
                <InputNumber
                  value={config.cooldown_period}
                  onChange={(value) =>
                    setConfig({ ...config, cooldown_period: value })
                  }
                  min={0}
                  max={86400}
                  step={300}
                  style={{ width: '100%' }}
                  suffix='秒'
                />
                <Text type='tertiary' size='small'>
                  默认: 3600 (1小时)
                </Text>
              </div>

              <div>
                <Text strong>时间恢复速率 (每小时恢复的百分比)</Text>
                <br />
                <InputNumber
                  value={config.time_recovery_rate}
                  onChange={(value) =>
                    setConfig({ ...config, time_recovery_rate: value })
                  }
                  min={1}
                  max={100}
                  step={5}
                  style={{ width: '100%' }}
                  suffix='%'
                />
                <Text type='tertiary' size='small'>
                  默认: 10 (10%/小时)
                </Text>
              </div>

              <div>
                <Text strong>最大连续失败次数</Text>
                <br />
                <InputNumber
                  value={config.max_consecutive_fails}
                  onChange={(value) =>
                    setConfig({ ...config, max_consecutive_fails: value })
                  }
                  min={1}
                  max={10}
                  step={1}
                  style={{ width: '100%' }}
                />
                <Text type='tertiary' size='small'>
                  默认: 4
                </Text>
              </div>
            </Space>
          )}
        </Modal>
      </div>
    </div>
  );
};

export default DynamicWeight;
