## L0
[English>>](#overview)
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
./lcnd
```

## License
L0遵循GPLv3协议发布。


### Overview
"L0" is a Distributed Ledger System launched by Beijing BoChen Technology Co., Ltd. (hereinafter referred to as "BoChen"). In L0, we pioneered the design of a tree-layered distributed ledger architecture. The architecture is a multi-chain organic combination, contains innovative cross-chain & hierarchical trading mechanisms and cross-chain consensus mechanisms, as well as a new classification of accounts, which breaks the traditional single-chain structure blockchain’s performance and storage bottlenecks. Theoretically it can support a network of any size and any level of concurrency, while providing real support for the resolution of hot accounts’ performance bottlenecks in realistic scenarios.

### Structure example
![lo](http://bocheninc.com/static/images/jiegou-en.png) 

### Overall Architecture
The entire architecture of L0 is divided into three layers: the core layer, the service layer and the application layer.

**The core layer** is composed of blockchain nodes and the message network and provides transaction broadcasting、consensus computing, contract execution、identity authentication、data storage and other functions in the ledger. The core layer can smoothly extend to improve trading processing capacity and elasticity. Enterprises can deploy their own core layers by themselves or access platform suppliers

**The service layer** can undertake various business scenarios based on the distributed ledger. Enterprises can utilize the service layer to build relevant specific business activities,including designing and submitting their own smart contract, establishing assets system, maintaining business and user data etc.

**The application layer** provides application services based on the distributed ledger to end users, such as wallets for various types of digital assets、transaction applications etc. The users can use the application layer to manage assets or make trades.

Please check [L0 whitepaper](http://bocheninc.com/l0.pdf) for more infomation

## Install
```
cd $GOPATH/src/github.com/bocheninc/L0/build
make
./lcnd
```
## License
L0 is distributed under the terms of the GPLv3 License.

