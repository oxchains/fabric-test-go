package sdkservice

import (
    "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
    "testing"
    "strconv"
)

const (
    basePath = "/home/ubuntu/gopath/src/github.com/hyperledger/fabric-manage-go/"
    configFilePath = "configFile.yaml"
    orgName = "Org1"
    orgAdmin = "Admin"
    channelID = "mychannel"
    channelConfigPath = "e2e_cli/channel-artifacts/channel.tx"
    ccID = "mycc"
    ccPath = "chaincode_example02"
    gopath = basePath + "/test/"
    version = "0"
)

var initArgs = [][]byte{[]byte("init"), []byte("a"), []byte("10"), []byte("b"), []byte("20")}
var invokeArgs = [][]byte{[]byte("a"), []byte("b"), []byte("5")}
var queryAArgs = [][]byte{[]byte("a")}
var queryBArgs = [][]byte{[]byte("b")}

func TestSDKService_CreateChannel(t *testing.T) {
    sdkService, err := NewSDKService(basePath + configFilePath)
    if err != nil {
        t.Fatalf("Get new sdkService failed")
    }
    
    orderOrg := "orderer"
    orderUser := "Admin"
    
    sdkService.Initialize(orderOrg, orderUser)
    
    err = sdkService.CreateChannel(orderOrg, orgName, basePath + channelConfigPath, channelID)
    if err != nil {
        t.Fatalf("CreateChannel failed")
    }
}

func TestSDKService_InstallCC(t *testing.T) {
    
    sdkService, err := NewSDKService(basePath + configFilePath)
    if err != nil {
        t.Fatalf("Get new sdkService failed")
    }
    
    orderOrg := "orderer"
    orderUser := "Admin"
    
    sdkService.Initialize(orderOrg, orderUser)
    
    err = sdkService.InstallCC(ccID, ccPath, gopath, version)
    if err != nil {
        t.Fatalf("Failed to install chaincode")
    }
}

func TestSDKService_InitializeCC(t *testing.T) {
    sdkService, err := NewSDKService(basePath + configFilePath)
    if err != nil {
        t.Fatalf("Get new sdkService failed")
    }
    
    orderOrg := "orderer"
    orderUser := "Admin"
    
    sdkService.Initialize(orderOrg, orderUser)
    
    ccPolicy := cauthdsl.SignedByAnyMember([]string{"Org1MSP"})
    err = sdkService.InitializeCC(channelID, ccID, ccPath, version, initArgs, ccPolicy)
    if err != nil {
        t.Fatalf("Failed to initializeCC")
    }
}

func TestSDKService_QueryCC(t *testing.T) {
    sdkService, err := NewSDKService(basePath + configFilePath)
    if err != nil {
        t.Fatalf("Get new sdkService failed")
    }
    
    orderOrg := "orderer"
    orderUser := "Admin"
    
    sdkService.Initialize(orderOrg, orderUser)
    
    value, err := sdkService.QueryCC(channelID, ccID, "query", queryAArgs)
    if err != nil {
        t.Fatalf("Failed to query ")
    }
    
    valueInt, _ := strconv.Atoi(string(value))
    if valueInt != 10 {
        t.Fatalf("Query reuslt is not equal")
    }
}

func TestSDKService_InvokeCC(t *testing.T) {
    sdkService, err := NewSDKService(basePath + configFilePath)
    if err != nil {
        t.Fatalf("Get new sdkService failed")
    }
    
    orderOrg := "orderer"
    orderUser := "Admin"
    
    sdkService.Initialize(orderOrg, orderUser)
    
    _, err = sdkService.InvokeCC(channelID, ccID, "invoke", invokeArgs)
    if err != nil {
        t.Fatalf("Failed to invoke transaction")
    }
}
