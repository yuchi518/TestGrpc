package main

import (
	pb "TestGrpc/my"
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials/oauth"
	"os"
	"path"

	//"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type GrpcDemoServer struct {
	pb.UnimplementedDemoServiceServer
}

func (e *GrpcDemoServer) Echo(ctx context.Context, req *pb.EchoRequest) (resp *pb.EchoReply, err error) {
	log.Infof("[Server] receive client request, client send:%s\n", req.Message)
	return &pb.EchoReply{
		Message:   req.Message,
		Timestamp: time.Now().Unix(),
	}, nil
}

func (e *GrpcDemoServer) Sum(stream pb.DemoService_SumServer) error {
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
	// set log environment
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	tlsServerCredentials, err := loadServerTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}
	tlsServerOption := grpc.Creds(tlsServerCredentials)

	oauthClientCredentials := oauth.NewOauthAccess(&oauth2.Token{AccessToken: "client-x-id"}) // TODO: create a jwt token
	tlsClientCredentials, err := loadClientTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}
	// insecureClientCredentials := grpc.WithTransportCredentials(insecure.NewCredentials())

	clientDialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(tlsClientCredentials),
		grpc.WithPerRPCCredentials(oauthClientCredentials),
	}
	//opts = append(opts, grpc.WithBlock())
	//tlsDialOption := grpc.WithTransportCredentials(tlsClientCredentials)

	go clientEcho(0*time.Second, clientDialOptions)
	go clientEcho(1*time.Second, clientDialOptions)
	go clientEcho(2*time.Second, clientDialOptions)
	go clientEcho(3*time.Second, clientDialOptions)
	go clientSum(1*time.Second, clientDialOptions)

	proxyDialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(tlsClientCredentials),
		//grpc.WithPerRPCCredentials(oauthClientCredentials),  // 如果 http gateway 不需要 token 驗證，則提供預設的 token
	}

	go createHttpReverseProxy(proxyDialOptions)

	runServerProcedure(tlsServerOption)
}

func clientEcho(waitingDuration time.Duration, opts []grpc.DialOption) {
	time.Sleep(waitingDuration)
	service := "Echo"

	conn, err := grpc.Dial(":9999", opts...)
	if err != nil {
		log.Fatalf("[Client/%s] 連線失敗：%v\n", service, err)
	}
	defer conn.Close()

	c := pb.NewDemoServiceClient(conn)
	log.Infof("[Client/%s] Ready\n", service)
	r, err := c.Echo(context.Background(), &pb.EchoRequest{Message: "HI HI HI HI"})
	if err != nil {
		log.Fatalf("[Client/%s] 無法執行 Plus 函式：%v\n", service, err)
	}

	log.Infof("[Client/%s] 回傳結果：%s , 時間:%s\n", service, r.Message, time.Unix(r.Timestamp, 0))
}

func clientSum(waitingDuration time.Duration, opts []grpc.DialOption) {
	time.Sleep(waitingDuration)
	service := "Sum"

	conn, err := grpc.Dial(":9999", opts...)
	if err != nil {
		log.Fatalf("[Client/%s] 連線失敗：%v\n", service, err)
	}
	defer conn.Close()

	c := pb.NewDemoServiceClient(conn)
	log.Infof("[Client/%s] Ready\n", service)

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
	log.Infof("[Client/%s] 結果 總和：%d 平均：%.2f\n", service, resp.Sum, resp.Average)
}

func runServerProcedure(tlsServerOption grpc.ServerOption) {
	apiListener, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Infoln(err)
		return
	}

	// 註冊 grpc
	es := &GrpcDemoServer{}

	// Intercept request to check the token.
	tokenValidationServerOption := grpc.UnaryInterceptor(tokenValidationHandler)
	// Enable TLS for all incoming connections.

	grpc := grpc.NewServer(tokenValidationServerOption, tlsServerOption)
	//pb.Re(grpc, es)
	pb.RegisterDemoServiceServer(grpc, es)
	reflection.Register(grpc)
	log.Infoln("[Server] running")
	if err := grpc.Serve(apiListener); err != nil {
		log.Fatal("[Server] grpc.Serve Error: ", err)
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

func tokenValidationHandler(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "missing metadata")
	}

	if !validAuthorizationHeader(md["authorization"]) {
		log.Errorf("Invalid authorization token: %v\n", md["authorization"])
		return nil, status.Errorf(codes.Unauthenticated, "invalid token")
	}

	return handler(ctx, req)
}

func validAuthorizationHeader(authorization []string) bool {
	if len(authorization) < 1 {
		return false
	}

	token := strings.TrimPrefix(authorization[0], "Bearer ")
	// TODO: jwt token validation
	return token == "client-x-id"
}

var (
// command-line options:
// gRPC server endpoint
//grpcServerEndpoint = flag.String("grpc-server-endpoint", "localhost:9999", "gRPC server endpoint")
)

func createOpenAPIHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ".swagger.json") {
			log.Errorf("Not Found: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		log.Infof("Serving %s", r.URL.Path)
		p := strings.TrimPrefix(r.URL.Path, "/openapiv2/")
		p = path.Join(dir, p)
		http.ServeFile(w, r, p)
	}
}

func CustomIncomingHeaderMatcher(key string) (string, bool) {
	//log.Infof("Header: %s\n", key)
	switch key {
	case "Authorization":
		//log.Infof("Matched\n")
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

func runHttpReverseProxy(opts []grpc.DialOption) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := http.NewServeMux()
	mux.HandleFunc("/openapiv2/", createOpenAPIHandler("my"))

	gw := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(CustomIncomingHeaderMatcher))
	err := pb.RegisterDemoServiceHandlerFromEndpoint(ctx, gw, ":9999" /*grpcServerEndpoint*/, opts)
	if err != nil {
		return err
	}

	mux.Handle("/", gw)

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	return http.ListenAndServe(":8081", mux)
}

func createHttpReverseProxy(opts []grpc.DialOption) {
	flag.Parse()

	if err := runHttpReverseProxy(opts); err != nil {
		log.Fatalf("[RESTFul] %v", err)
	}
}
