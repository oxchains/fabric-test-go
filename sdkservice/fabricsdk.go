package sdkservice

import (
    "github.com/hyperledger/fabric-sdk-go/def/fabapi"
    fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
    chmgmt "github.com/hyperledger/fabric-sdk-go/api/apitxn/chmgmtclient"
    packager "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/ccpackager/gopackager"
    resmgmt "github.com/hyperledger/fabric-sdk-go/api/apitxn/resmgmtclient"
    cb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
    
    "time"
    "github.com/hyperledger/fabric-sdk-go/api/apitxn"
)

const (
    ADMIN = "Admin"
    USER = "User"
)

// Supply service of connecting fabric peer/orderer to execute operations
// sdk; channel; chaincode; user; org;
type SDKService struct {
    Options fabapi.Options
    SDK *fabapi.FabricSDK
    syClient fab.FabricClient
    DefaultResMgmtClient resmgmt.ResourceMgmtClient
}

// Get new sdk service instance
func NewSDKService(configFile string) (*SDKService, error) {
    sdkOptions := fabapi.Options{
        ConfigFile: configFile,
    }
    
    return &SDKService{
        Options: sdkOptions,
    }, nil
}

// Initialize sdk service
// Use org and user to setup sdk
func (sdkService *SDKService) Initialize(orgID, user string) error {
    
    sdk, err := fabapi.NewSDK(sdkService.Options)
    if err != nil {
        return err
    }
    
    session, err := sdkService.SDK.NewPreEnrolledUserSession(orgID, user)
    if err != nil {
        return err
    }
    
    sc, err := sdk.NewSystemClient(session)
    if err != nil {
        return err
    }
    
    sdkService.syClient = sc
    
    return nil
}

// Create channel and will let org join channel automatical
func (sdkService *SDKService) CreateChannel(orderOrgName, orgName, channelConfig, channelID string) error {
    
    chMgmtClient, err := sdkService.SDK.NewChannelMgmtClientWithOpts(ADMIN, &fabapi.ChannelMgmtClientOpts{OrgName: orderOrgName})
    if err != nil {
        return err
    }
    
    // Org admin user is signing user for creating channel
    orgAdminUser, err := sdkService.SDK.NewPreEnrolledUser(orgName, ADMIN)
    if err != nil {
        return err
    }
    
    req := chmgmt.SaveChannelRequest{ChannelID: channelID, ChannelConfig: channelConfig, SigningUser: orgAdminUser}
    if err = chMgmtClient.SaveChannel(req); err != nil {
        return err
    }
    
    // TODO change to listen event
    time.Sleep(time.Second * 3)
    
    // Resource management client is responsible for managing resources (joining channels, install/instantiate/upgrade chaincodes)
    resMgmtClient, err := sdkService.SDK.NewResourceMgmtClient(ADMIN)
    if err != nil {
        return err
    }
    
    if err = resMgmtClient.JoinChannel(channelID); err != nil {
        return err
    }
    
    sdkService.DefaultResMgmtClient = resMgmtClient
    
    return nil
}

// TODO update channel config
func (sdkService *SDKService) UpdateChannel() {

}

func (sdkService *SDKService) InstallCC(ccID, ccPath, gopath, version string) error {
    // Create chaincode package for example cc
    // final chaincode path = gopath + "/src/" + ccPath
    ccPkg, err := packager.NewCCPackage(ccPath, gopath)
    if err != nil {
        return err
    }
    
    // Install example cc to org peers
    installCCReq := resmgmt.InstallCCRequest{Name: ccID, Path: ccPath, Version: version, Package: ccPkg}
    _, err = sdkService.DefaultResMgmtClient.InstallCC(installCCReq)
    if err != nil {
        return err
    }
    
    return nil
}

func (sdkService *SDKService) InitializeCC(channelID, ccID, ccPath, version string, args [][]byte, ccPolicy *cb.SignaturePolicyEnvelope) error {
    
    err := sdkService.DefaultResMgmtClient.InstantiateCC(channelID, resmgmt.InstantiateCCRequest{Name: ccID, Path: "chaincode_example02", Version: version, Args: args, Policy: ccPolicy})
    if err != nil {
        return err
    }
    
    return nil
}

func (sdkService *SDKService) QueryCC(channelID, ccID, fcn string, args [][]byte) ([]byte, error) {
    // Channel client is used to query and execute transactions
    chClient, err := sdkService.SDK.NewChannelClient(channelID, "User1")
    if err != nil {
        return []byte{}, err
    }
    
    value, err := chClient.Query(apitxn.QueryRequest{ChaincodeID: ccID, Fcn: fcn, Args: args})
    if err != nil {
        return []byte{}, err
    }
    
    return value, nil
}

func (sdkService *SDKService) InvokeCC(channelID, ccID, fcn string, args [][]byte) (apitxn.TransactionID, error) {
    
    chClient, err := sdkService.SDK.NewChannelClient(channelID, "User1")
    if err != nil {
        return apitxn.TransactionID{}, err
    }
    
    value, err := chClient.ExecuteTx(apitxn.ExecuteTxRequest{ChaincodeID: ccID, Fcn: fcn, Args: args})
    if err != nil {
        return apitxn.TransactionID{}, err
    }
    
    return value, nil
}


