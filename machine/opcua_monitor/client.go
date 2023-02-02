package opcua_monitor

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/doteich/OPC-UA-Logger/setup"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/monitor"
	"github.com/gopcua/opcua/ua"
)

func CreateOPCUAMonitor(config setup.Config) {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-signalCh
		println()
		cancel()
	}()

	ep := ValidateEndpoint(ctx, config.ClientConfig.Url, config.ClientConfig.SecurityPolicy, config.ClientConfig.SecurityMode)

	connectionParams := SetClientOptions(&config, ep)

	client := CreateClientConnection(config.ClientConfig.Url, connectionParams)
	err := client.Connect(ctx)

	if err != nil {
		fmt.Println(err)
	}

	defer client.CloseSessionWithContext(ctx)

	nodeMonitor, err := monitor.NewNodeMonitor(client)

	if err != nil {
		panic("Failed to setup monitor")
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go MonitorItems(ctx, nodeMonitor, time.Duration(config.LoggerConfig.Interval*1000000000), 1000, wg, config.Nodes)

	<-ctx.Done()
	defer ShowDone()
}

func ShowDone() {
	fmt.Println("ABORTING")

}

func ValidateEndpoint(ctx context.Context, endpoint string, policy string, mode string) *ua.EndpointDescription {
	endpoints, err := opcua.GetEndpoints(ctx, endpoint)

	if err != nil {
		fmt.Println(endpoint + policy + mode)
		errorHandler(err)
	}

	ep := opcua.SelectEndpoint(endpoints, policy, ua.MessageSecurityModeFromString(mode))

	if ep == nil {
		panic("No Matching Endpoint Found - Check Configuration")
	}

	return ep

}

func SetClientOptions(config *setup.Config, ep *ua.EndpointDescription) []opcua.Option {

	// basic params
	connectionParams := []opcua.Option{
		opcua.SecurityPolicy(config.ClientConfig.SecurityPolicy),
		opcua.SecurityModeString(config.ClientConfig.SecurityMode),
	}

	if config.ClientConfig.SecurityMode != "None" || config.ClientConfig.SecurityPolicy != "None" {
		connectionParams = append(connectionParams, opcua.CertificateFile("./certs/cert.pem"))
		connectionParams = append(connectionParams, opcua.PrivateKeyFile("./certs/private_key.pem"))
	}

	switch config.ClientConfig.AuthType {
	case "User & Password":
		connectionParams = append(connectionParams, opcua.AuthUsername(config.ClientConfig.Username, config.ClientConfig.Password))
		connectionParams = append(connectionParams, opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeUserName))
	case "Certificate":
		//connectionParams = append(connectionParams, opcua.AuthCertificate())
		connectionParams = append(connectionParams, opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeCertificate))
	default:
		connectionParams = append(connectionParams, opcua.AuthAnonymous())
		connectionParams = append(connectionParams, opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous))
	}

	return connectionParams
}

func CreateClientConnection(ep string, options []opcua.Option) *opcua.Client {

	return opcua.NewClient(ep, options...)

}
