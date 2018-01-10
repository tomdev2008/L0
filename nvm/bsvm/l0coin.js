/***** 用合约来完成一个数字货币系统 *****/

// 合约创建时会被调用一次，之后就不会被调用
function L0Init(args) {
    console.log("===L0Init Js++++")
    // console.log(args[0], args[1])
    // var account = L0.Account();
    //console.log("===========111==============");
    //console.log(JSON.stringify({name:"Tim",number:12345}));
    L0.PutState("hello", JSON.stringify({name:"Tim",number:12345}));
    //console.log("===========222==============");
    console.log(">>>:", L0.GetState("hello"));
    L0.PutState("Value", 0);
    return true;
}

// 每次合约执行都调用
function L0Invoke(func, args) {
    // var receiver = args[0];
    // var amount = args[1];
    // console.log("==>>>> ", L0.GetState("hello"))
    //
    // if("testwrite" == func) {
    //     return testwrite();
    // }
    console.log("invoke js")
    L0.Sleep(1)
    var info = 5;//L0.GetState("Value");
    var cnt = 0;
    while(cnt < 100) {
        cnt ++;
        L0.PutState(args[0]+cnt, args[0]+cnt)
        //console.log(args[0]+cnt);
    }
    //console.log("invoke", args[0])
    info ++;
    if (info % 1000 == 0) {
        //console.log("======>>> ", info);
    }
    L0.PutState("Value", info);

    return true;
}

function L0Query(args) {
    console.log("call L0Query");
    return "query ok"
}


function testwrite() {
    // console.log("start testwrite ...");
    // L0.PutState("a1", JSON.stringify("a1"));
    // L0.PutState("a2", JSON.stringify(true));
    // var m = new Object();
    // m["key1"] = "Comtop0";
    // m["key2"] = "Comtop1";
    // m["key3"] = "Comtop2";
    // L0.PutState("a3", JSON.stringify(m));
    // return true
}

function testread() {

}
