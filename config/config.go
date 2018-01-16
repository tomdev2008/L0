// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of L0
//
// The L0 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The L0 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package config

import (
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/merge"
	"github.com/bocheninc/L0/core/p2p"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/rpc"
	"github.com/bocheninc/L0/vm"
	"github.com/bocheninc/base/log"
	"github.com/spf13/viper"
)

const (
	defaultConfigFilename       = "lcnd.yaml"
	defaultVMLogFilename        = "vm.log"
	defaultLogFilename          = "lcnd.log"
	defaultChainDataDirname     = "chaindata"
	defaultLogDirname           = "logs"
	defaultKeyStoreDirname      = "keystore"
	defaultNodeDirname          = "node"
	defaultNodeKeyFilename      = "nodekey"
	defaultPluginDirname        = "plugin"
	defaultExceptinBlockDirname = "except_write_block"
	defaultMaxPeers             = 8
)

var (
	defaultConfig = &Config{
		NetConfig:   p2p.DefaultConfig(),
		DbConfig:    db.DefaultConfig(),
		MergeConfig: merge.DefaultConfig(),

		LogLevel: "debug",
		LogFile:  defaultLogFilename,
		LogFormatter: &logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000",
			FullTimestamp:   true,
		},
	}

	privkey *crypto.PrivateKey
)

// Config Represents the global config of lcnd
type Config struct {
	// dir
	DataDir                string
	LogDir                 string
	NodeDir                string
	KeyStoreDir            string
	PluginDir              string
	WriteExceptionBlockDir string

	// file
	PeersFile  string
	ConfigFile string

	// net
	NetConfig *p2p.Config

	// Merger
	MergeConfig *merge.Config

	// log
	LogLevel     string
	LogFile      string
	LogFormatter logrus.Formatter

	// db
	DbConfig    *db.Config
	NetDbConfig *db.Config

	//jrpc
	RpcConfig *rpc.Config

	// profile
	CPUFile  string
	ProfPort string
}

// New returns a config according the config file
func New(cfgFile string) (cfg *Config, err error) {
	return loadConfig(cfgFile)
}

func loadConfig(cfgFile string) (conf *Config, err error) {
	var (
		cfg        *Config
		appDataDir string
	)

	cfg = defaultConfig

	if cfgFile != "" {
		if utils.FileExist(cfgFile) {
			viper.SetConfigFile(cfgFile)
		}
		if err := viper.ReadInConfig(); err != nil {
			log.Debugf("no config file, run as default config! viper.ReadInConfig error %s", err)
		} else {
			appDataDir = cfg.read()
		}
	}

	if appDataDir == "" {
		appDataDir = utils.AppDataDir()
		cfgFile = filepath.Join(appDataDir, defaultConfigFilename)
		if utils.FileExist(cfgFile) {
			viper.SetConfigFile(cfgFile)
			if err := viper.ReadInConfig(); err != nil {
				log.Debug("no config file, run as default config!")
			} else {
				if dir := cfg.read(); dir != "" {
					if ok, _ := utils.IsDirExist(dir); ok {
						appDataDir = dir
					}
				}
			}
		}
	}

	utils.OpenDir(appDataDir)

	cfg.DataDir, err = utils.OpenDir(filepath.Join(appDataDir, defaultChainDataDirname))
	cfg.LogDir, err = utils.OpenDir(filepath.Join(appDataDir, defaultLogDirname))
	cfg.KeyStoreDir, err = utils.OpenDir(filepath.Join(appDataDir, defaultKeyStoreDirname))
	cfg.NodeDir, err = utils.OpenDir(filepath.Join(appDataDir, defaultNodeDirname))
	cfg.PluginDir, err = utils.OpenDir(filepath.Join(appDataDir, defaultPluginDirname))
	cfg.WriteExceptionBlockDir, err = utils.OpenDir(filepath.Join(appDataDir, defaultExceptinBlockDirname))

	/*set chainid from config file just for test*/
	cfg.readParamConfig()

	cfg.DbConfig = DBConfig(cfg.DataDir)
	cfg.NetDbConfig = DBConfig(cfg.NodeDir)
	cfg.NetConfig = NetConfig(cfg.NodeDir)
	cfg.MergeConfig = MergeConfig(cfg.NodeDir)
	cfg.readLogConfig()
	vm.VMConf = VMConfig(cfg.LogFile, cfg.LogLevel)
	cfg.RpcConfig = JrpcConfig(cfg.LogFile, cfgFile)
	return cfg, nil
}

func (cfg *Config) read() string {
	var (
		dataDir  string
		cpuFile  string
		profPort string
	)

	if profPort = viper.GetString("blockchain.profPort"); profPort != "" {
		cfg.ProfPort = profPort
	}

	if cpuFile = viper.GetString("blockchain.cpuprofile"); cpuFile != "" {
		cfg.CPUFile = cpuFile
	}
	if dataDir = viper.GetString("blockchain.datadir"); dataDir != "" {
		return dataDir
	}

	return dataDir
}

/*set chainid from config file just for test*/
func (cfg *Config) readParamConfig() {
	str := getString("blockchain.chainId", "NET_NOT_SET")
	pk := getStringSlice("issueaddr.addr", []string{})
	params.ChainID = utils.HexToBytes(str)
	params.PublicAddress = pk

	params.MaxOccurs = getInt("blockchain.maxOccurs", 1)

	nodeType := getString("blockchain.nodeType.type", "vp")
	if nodeType == "vp" {
		params.Nvp = false
	} else {
		params.Nvp = true
		params.Mongodb = getbool("blockchain.nodeType.mongodb", false)
	}
}

func (cfg *Config) readLogConfig() {
	var (
		logLevel, logFile, logFormatter string
	)
	if logLevel = viper.GetString("log.level"); logLevel != "" {
		cfg.LogLevel = logLevel
	}
	if logFile = filepath.Join(cfg.LogDir, defaultLogFilename); logFile != "" {
		cfg.LogFile = logFile
	}

	if logFormatter = viper.GetString("log.formatter"); logFormatter != "" {
		if logFormatter == "json" {
			cfg.LogFormatter = &logrus.JSONFormatter{
				TimestampFormat: "2006-01-02 15:04:05.000",
			}
		}
	}

}
