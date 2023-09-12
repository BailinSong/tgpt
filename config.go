package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ConfigManager 结构体用于管理配置文件
type ConfigManager struct {
	configFile string
}

// NewConfigManager 创建一个新的 ConfigManager 实例
func NewConfigManager(configFile string) *ConfigManager {
	return &ConfigManager{
		configFile: configFile,
	}
}

// ReadConfig 读取配置文件，如果文件不存在则创建并返回默认配置
func (cm *ConfigManager) ReadConfig(defaultConfig map[string]interface{}) (map[string]interface{}, error) {
	data, err := os.ReadFile(cm.configFile)

	if err != nil {
		if os.IsNotExist(err) {
			// 如果文件不存在，则创建并写入默认配置数据
			err := cm.writeDefaultConfig(defaultConfig)
			if err != nil {
				return nil, err
			}
			return defaultConfig, nil
		}
		return nil, err
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		return nil, err
	}

	return configData, nil
}

// WriteConfig 将配置数据写入配置文件
func (cm *ConfigManager) WriteConfig(configData map[string]interface{}) error {
	configDataJSON, err := json.MarshalIndent(configData, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile(cm.configFile, configDataJSON, 0644)
	if err != nil {
		return err
	}

	return nil
}

// writeDefaultConfig 写入默认配置数据到文件
func (cm *ConfigManager) writeDefaultConfig(defaultConfig map[string]interface{}) error {
	configFileDir := filepath.Dir(cm.configFile)
	if err := os.MkdirAll(configFileDir, 0755); err != nil {
		return err
	}

	configDataJSON, err := json.MarshalIndent(defaultConfig, "", "    ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(cm.configFile, configDataJSON, 0644); err != nil {
		return err
	}

	return nil
}
