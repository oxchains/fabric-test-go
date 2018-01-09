package sdkservice

import (
	fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	chmgmt "github.com/hyperledger/fabric-sdk-go/api/apitxn/chmgmtclient"
	resmgmt "github.com/hyperledger/fabric-sdk-go/api/apitxn/resmgmtclient"
	"github.com/hyperledger/fabric-sdk-go/def/fabapi"
	packager "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/ccpackager/gopackager"
	cb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
    "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/events"
    "github.com/hyperledger/fabric-sdk-go/pkg/errors"
    pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
    "github.com/hyperledger/fabric-sdk-go/pkg/logging"

	"time"

	"github.com/hyperledger/fabric-sdk-go/api/apitxn"
    "fmt"
)

var logger = logging.NewLogger("sdkservice")

const (
	ADMIN = "Admin"
	USER  = "User"
)

// Supply service of connecting fabric peer/orderer to execute operations
// sdk; channel; chaincode; user; org;
type SDKService struct {
	Options              fabapi.Options
	SDK                  *fabapi.FabricSDK
	syClient             fab.FabricClient
	DefaultResMgmtClient resmgmt.ResourceMgmtClient
	EventHub             fab.EventHub
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

	sdkService.SDK = sdk

	session, err := sdkService.SDK.NewPreEnrolledUserSession(orgID, user)
	if err != nil {
		return err
	}

	sc, err := sdk.NewSystemClient(session)
	if err != nil {
		return err
	}
	sdkService.syClient = sc
	
	if err := sdkService.setupEventHub(); err != nil {
	    return err
    }

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
	// Save channel not return txID
	if err = chMgmtClient.SaveChannel(req); err != nil {
		return err
	}
	
	// Wait for creating channel
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

	if sdkService.DefaultResMgmtClient == nil {
		err := sdkService.SetDefaultResMgmtClient()
		if err != nil {
			return err
		}
	}
	_, err = sdkService.DefaultResMgmtClient.InstallCC(installCCReq)
	if err != nil {
		return err
	}

	return nil
}

func (sdkService *SDKService) InitializeCC(channelID, ccID, ccPath, version string, args [][]byte, ccPolicy *cb.SignaturePolicyEnvelope) error {

	if sdkService.DefaultResMgmtClient == nil {
		err := sdkService.SetDefaultResMgmtClient()
		if err != nil {
			return err
		}
	}

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

// Invoke transaction with event to make sure the transaction is right
// executed or not
func (sdkService *SDKService) InvokeCC(channelID, ccID, fcn string, args [][]byte) (apitxn.TransactionID, bool, error) {

	chClient, err := sdkService.SDK.NewChannelClient(channelID, "User1")
	if err != nil {
		return apitxn.TransactionID{}, false, err
	}
 
	txRes := make(chan apitxn.ExecuteTxResponse)
    txOpts := apitxn.ExecuteTxOpts{Notifier: txRes}
    
	_, err = chClient.ExecuteTxWithOpts(apitxn.ExecuteTxRequest{ChaincodeID: ccID, Fcn: fcn, Args: args}, txOpts)
	if err != nil {
		return apitxn.TransactionID{}, false, err
	}
	
	select {
	case res := <- txRes:
	   if res.Error != nil {
	       return res.Response, false, res.Error
       } else {
           fmt.Println(res.TxValidationCode)
           return res.Response, true, nil
       }
    }
}

func (sdkService *SDKService) InvokeCCAsync(channelID, ccID, fcn string, args [][]byte) (apitxn.TransactionID, error) {
    
    chClient, err := sdkService.SDK.NewChannelClient(channelID, "User1")
    if err != nil {
        return apitxn.TransactionID{}, err
    }
    
    txnID, err := chClient.ExecuteTx(apitxn.ExecuteTxRequest{ChaincodeID: ccID, Fcn: fcn, Args: args})
    if err != nil {
        return apitxn.TransactionID{}, err
    }
    
    return txnID, nil
}

func (sdkService *SDKService) setupEventHub() error {
    eventHub, err := sdkService.getEventHub()
    if err != nil {
        return err
    }
    
    if err := eventHub.Connect(); err != nil {
        return errors.WithMessage(err, "eventHub connect failed")
    }
    sdkService.EventHub = eventHub
    
    return nil
}

func (sdkService *SDKService) getEventHub() (fab.EventHub, error) {
    
    session, err := sdkService.SDK.NewPreEnrolledUserSession("org1", "Admin")
    if err != nil {
        return nil, err
    }
    
    sc, err := sdkService.SDK.NewSystemClient(session)
    if err != nil {
        return nil, err
    }
    
    eventHub, err := events.NewEventHub(sc)
    if err != nil {
        return nil, errors.WithMessage(err, "NewEventHub failed")
    }
    
    foundEventHub := false
    peerConfig, err := sdkService.syClient.Config().PeersConfig("org1")
    if err != nil {
        return nil, errors.WithMessage(err, "PeersConfig failed")
    }
    
    for _, p := range peerConfig {
        if p.URL != "" {
            //("EventHub connect to peer (%s)", p.URL)
            serverHostOverride := ""
            if str, ok := p.GRPCOptions["ssl-target-name-override"].(string); ok {
                serverHostOverride = str
            }
            eventHub.SetPeerAddr(p.EventURL, p.TLSCACerts.Path, serverHostOverride)
            foundEventHub = true
            break
        }
    }
    
    if !foundEventHub {
        return nil, errors.New("event hub configuration not found")
    }
    
    return eventHub, nil
}

// RegisterTxEvent registers on the given eventhub for the give transaction
// returns a boolean channel which receives true when the event is complete
// and an error channel for errors
func (sdkService *SDKService) registerTxEvent(txID apitxn.TransactionID) (chan bool, chan error) {
    done := make(chan bool)
    fail := make(chan error)
    
    // This may happended after the exec of transaction, and will never get
    // Change to use executeTxWithOpts
    sdkService.EventHub.RegisterTxEvent(txID, func(txId string, errorCode pb.TxValidationCode, err error) {
        if err != nil {
            logger.Debugf("Received error event for txid(%s)", txId)
            fail <- err
        } else {
            logger.Debugf("Received success event for txid(%s)", txId)
            done <- true
        }
    })
    
    return done, fail
}

func (sdkService *SDKService) GetDefaultResMgmtClient() (resmgmt.ResourceMgmtClient, error) {

	resMgmtClient, err := sdkService.SDK.NewResourceMgmtClient(ADMIN)
	if err != nil {
		return nil, err
	}

	return resMgmtClient, nil
}

func (sdkService *SDKService) SetDefaultResMgmtClient() error {

	client, err := sdkService.GetDefaultResMgmtClient()
	if err != nil {
		return err
	}

	sdkService.DefaultResMgmtClient = client

	return nil
}
