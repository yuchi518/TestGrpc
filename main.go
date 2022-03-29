package main

import (
	pb "TestGrpc/my"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"io"
	"io/ioutil"
	"log"
	"net"
	"time"
)

type EchoServer struct {
	pb.DemoServiceServer
}

func (e *EchoServer) Echo(ctx context.Context, req *pb.EchoRequest) (resp *pb.EchoReply, err error) {
	log.Printf("[Server] receive client request, client send:%s\n", req.Message)
	return &pb.EchoReply{
		Message:   req.Message,
		Timestamp: time.Now().Unix(),
	}, nil
}

func (e *EchoServer) Sum(stream pb.DemoService_SumServer) error {
	var total int64 = 0
	var count int64 = 0
	for {
		sumReq, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.SumResponse{Average: float64(total) / float64(count), Sum: total})
		}
		if err != nil {
			return err
		}

		count++
		total += sumReq.Value
	}
}

func main() {
	tlsServerCredentials, err := loadServerTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}
	tlsServerOption := grpc.Creds(tlsServerCredentials)

	tlsClientCredentials, err := loadClientTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}
	// insecureClientCredentials := grpc.WithTransportCredentials(insecure.NewCredentials())

	tlsDialOption := grpc.WithTransportCredentials(tlsClientCredentials)

	go clientEcho(0*time.Second, tlsDialOption)
	go clientEcho(1*time.Second, tlsDialOption)
	go clientEcho(2*time.Second, tlsDialOption)
	go clientEcho(3*time.Second, tlsDialOption)
	go clientSum(1*time.Second, tlsDialOption)
	serverProcedure(tlsServerOption)
}

func clientEcho(waitingDuration time.Duration, tlsDialogOption grpc.DialOption) {
	time.Sleep(waitingDuration)
	service := "Echo"

	conn, err := grpc.Dial(":9999", tlsDialogOption)
	if err != nil {
		log.Fatalf("[Client/%s] 連線失敗：%v\n", service, err)
	}
	defer conn.Close()

	c := pb.NewDemoServiceClient(conn)
	log.Printf("[Client/%s] Ready\n", service)
	r, err := c.Echo(context.Background(), &pb.EchoRequest{Message: "HI HI HI HI"})
	if err != nil {
		log.Fatalf("[Client/%s] 無法執行 Plus 函式：%v\n", service, err)
	}

	log.Printf("[Client/%s] 回傳結果：%s , 時間:%s\n", service, r.Message, time.Unix(r.Timestamp, 0))
}

func clientSum(waitingDuration time.Duration, tlsDialogOption grpc.DialOption) {
	time.Sleep(waitingDuration)
	service := "Sum"

	conn, err := grpc.Dial(":9999", tlsDialogOption)
	if err != nil {
		log.Fatalf("[Client/%s] 連線失敗：%v\n", service, err)
	}
	defer conn.Close()

	c := pb.NewDemoServiceClient(conn)
	log.Printf("[Client/%s] Ready\n", service)

	stream, err := c.Sum(context.Background())
	if err != nil {
		log.Fatalf("[Client/%s] 準備sum串流失敗：%v\n", service, err)
	}
	for i := 0; i < 10; i++ {
		err = stream.Send(&pb.SumRequest{Value: int64(i)})
		if err != nil {
			log.Fatalf("[Client/%s] 傳送sum值失敗：%v\n", service, err)
		}
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("[Client/%s] 結束sum串流並取回結果失敗：%v\n", service, err)
	}
	log.Printf("[Client/%s] 結果 總和：%d 平均：%.2f\n", service, resp.Sum, resp.Average)
}

func serverProcedure(tlsServerOption grpc.ServerOption) {
	apiListener, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Println(err)
		return
	}

	// 註冊 grpc
	es := &EchoServer{}

	grpc := grpc.NewServer(tlsServerOption)
	//pb.Re(grpc, es)
	pb.RegisterDemoServiceServer(grpc, es)
	reflection.Register(grpc)
	log.Println("[Server] running")
	if err := grpc.Serve(apiListener); err != nil {
		log.Fatal(" [Server] grpc.Serve Error: ", err)
		return
	}
}

// loadServerTLSCredentials
func loadServerTLSCredentials() (credentials.TransportCredentials, error) {
	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair("cert/server-cert.pem", "cert/server-key.pem")
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}

	return credentials.NewTLS(config), nil
}

func loadClientTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := ioutil.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Create the credentials and return it
	config := &tls.Config{
		RootCAs: certPool,
	}

	return credentials.NewTLS(config), nil
}
