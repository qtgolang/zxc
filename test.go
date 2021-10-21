package zxc

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
}

func (handler *Handler) BeforeRequest(entity *Entity) {
	entity.Request.Header.Set("Accept-Encoding", "")

	Mod := entity.Request.Method
	Host := entity.Request.Host
	Path := entity.Request.RequestURI
	buf := new(bytes.Buffer)
	buf.ReadFrom(entity.GetRequestBody())
	Body := buf.String()

	fmt.Println("请求 Mod", Mod)
	fmt.Println("请求 Host", Host)
	fmt.Println("请求 Path", Path)
	fmt.Println("请求 Body len", len(Body))
	fmt.Println("请求 Body", Body)

	//qt.Call(callback, Mod, Host, Path, Body, len(Body))

}
func (handler *Handler) BeforeResponse(entity *Entity, err error) {
	Mod := entity.Request.Method
	Host := entity.Request.Host
	Path := entity.Request.RequestURI
	buf := new(bytes.Buffer)
	buf.ReadFrom(entity.GetResponseBody())
	Body := buf.String()

	fmt.Println("Ret Mod", Mod)
	fmt.Println("Ret Host", Host)
	fmt.Println("Ret Path", Path)
	fmt.Println("Ret Body len", len(Body))
	fmt.Println("Ret Body", Body)
	//go qt.Call(callback, Mod, Host, Path, Body, len(Body))
}
func (handler *Handler) ErrorLog(err error) {}

func testmain() {
	//使用默认的CA 证书 客户端需要手动把ca 证书安装到信任的证书列表
	Stat(8899, &Handler{}, RootCa, RootKey)
}

func Stat(prot int, delegate Delegate, certCa, certKey string) {
	proxy := NewWithDelegate(delegate, certCa, certKey)
	server := &http.Server{
		Addr: ":" + strconv.Itoa(prot),
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			proxy.ServerHandler(rw, req)
		}),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}
