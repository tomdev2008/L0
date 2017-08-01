## L0
[English>>](https://github.com/bocheninc/L0/blob/master/README-en.md)

[![Build Status](https://travis-ci.org/bocheninc/L0.svg?branch=master)](https://travis-ci.org/bocheninc/L0)
[![GoDoc](https://godoc.org/github.com/bocheninc/L0?status.svg)](https://godoc.org/github.com/bocheninc/L0)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/Bochen-L0/Lobby)

### 概述

L0是北京博晨技术有限公司推出的的具有自主知识产权的分布式账本系统。L0使用了树型分层的分布式账本架构,它是多个链的有机组合，通过创新的跨链共识和分层交易机制，以及对账户和交易的全新分类，突破了传统单链结构的性能及存储瓶颈，理论上可支持任意规模的网络和任意级别的并发，同时为现实场景中热点账户性能瓶颈的解决提供了内生的支持。

### 结构示例

![L0三层结构示例](http://bocheninc.com/static/images/jiegou.jpg)

### 整体架构

L0整体架构分为三层：核心层、服务层、应用层。

**核心层**由区块链节点和消息网络组成，提供账本的交易广播、共识计算、合约执行、身份认证、数据存储等功能。核心层
可以通过弹性平滑扩展提升交易处理能力。企业可以快速部署自有的核心层，也可以接入基础平台供应商的核心层。

**服务层**可以承接分布式账本的各种业务场景，企业可以通过服务层构建相关的具体业务，包括在服务层构建和提交自己的
智能合约，构建自己的资产体系，维护自己的业务数据、用户数据等。

**应用层**向终端用户提供基于分布式账本的应用服务，如各类型数字资产的钱包、交易应用等。用户通过应用层来管理资产
或者进行交易。

更多关于L0的介绍请查看[L0白皮书](http://bocheninc.com/l0.pdf)

## 安装

```
cd $GOPATH/src/github.com/bocheninc/L0/build
make
bash start.sh
```

## 许可证

L0遵循GPLv3协议发布。
