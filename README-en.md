## L0

[English>>](https://github.com/bocheninc/L0/blob/master/README-en.md)

[![Build Status](https://travis-ci.org/bocheninc/L0.svg?branch=master)](https://travis-ci.org/bocheninc/L0)
[![GoDoc](https://godoc.org/github.com/bocheninc/L0?status.svg)](https://godoc.org/github.com/bocheninc/L0)

### Overview

"L0" is a Distributed Ledger System launched by Beijing BoChen Technology Co., Ltd. (hereinafter referred to as "BoChen"). In L0, we pioneered the design of a tree-layered distributed ledger architecture. The architecture is a multi-chain organic combination, contains innovative cross-chain & hierarchical trading mechanisms and cross-chain consensus mechanisms, as well as a new classification of accounts, which breaks the traditional single-chain structure blockchain’s performance and storage bottlenecks. Theoretically it can support a network of any size and any level of concurrency, while providing real support for the resolution of hot accounts’ performance bottlenecks in realistic scenarios.

### Structure example

![lo](http://bocheninc.com/static/images/jiegou-en.png) 

### Overall Architecture

The entire architecture of L0 is divided into three layers: the core layer, the service layer and the application layer.

**The core layer** is composed of blockchain nodes and the message network and provides transaction broadcasting、consensus computing, contract execution、identity authentication、data storage and other functions in the ledger. The core layer can smoothly extend to improve trading processing capacity and elasticity. Enterprises can deploy their own core layers by themselves or access platform suppliers

**The service layer** can undertake various business scenarios based on the distributed ledger. Enterprises can utilize the service layer to build relevant specific business activities,including designing and submitting their own smart contract, establishing assets system, maintaining business and user data etc.

**The application layer** provides application services based on the distributed ledger to end users, such as wallets for various types of digital assets、transaction applications etc. The users can use the application layer to manage assets or make trades.

Please check [L0 whitepaper](http://bocheninc.com/l0-en.pdf) for more infomation

## Install

```
cd $GOPATH/src/github.com/bocheninc/L0/build
make
bash start.sh
```
## License

L0 is distributed under the terms of the GPLv3 License.

