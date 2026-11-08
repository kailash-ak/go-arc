package main
import (
	"log"
	"fmt"
	"net/http"
	"io"
	"encoding/json"
)
func homeHandler(w http.ResponseWriter, r *http.Request){
	fmt.Fprintln(w,"home")
}
func healthHandler(w http.ResponseWriter,r *http.Request){
	w.Header().Set("Content-Type","application/json")
	fmt.Fprintf(w,`{"status":"ok"}`)
}
func echoHandler(w http.ResponseWriter, r *http.Request){
	body,err:=io.ReadAll(r.Body)
	if err!=nil{
		log.Println(err)
		http.Error(w,"Internal Server Error",http.StatusInternalServerError)
		return
	}
	w.Write(body)
}
func userHandler(w http.ResponseWriter, r *http.Request){
	id:=r.PathValue("id")
	fmt.Fprintf(w,"user id: %s",id)
}
func greetHandler(w http.ResponseWriter, r *http.Request){

	type RequestBody struct{
		Name string `json:"name"`
	}
	type ResponseBody struct{
		Message string `json:"message"`
	}
	var v RequestBody
	err:=json.NewDecoder(r.Body).Decode(&v)
	if err!=nil{
		log.Printf("JSON decode error: %v", err)
		http.Error(w,"Bad Request",http.StatusBadRequest)
		return
	}
	response:=ResponseBody{
		Message: fmt.Sprintf("Hello, %s!",v.Name),
	}
	w.Header().Set("Content-Type","application/json")
	json.NewEncoder(w).Encode(response)
}
func testHandler(w http.ResponseWriter, r *http.Request){
	w.Write([]byte("test"))
	w.WriteHeader(http.StatusNotFound)
}

func main(){
	mux:=http.NewServeMux()
	mux.HandleFunc("GET /{$}",homeHandler)
	mux.HandleFunc("GET /health",healthHandler)
	mux.HandleFunc("POST /echo", echoHandler)
	mux.HandleFunc("GET /users/{id}",userHandler)
	mux.HandleFunc("POST /greet",greetHandler)
	mux.HandleFunc("GET /test", testHandler)
	srv:= &http.Server{
		Addr: ":8080",
		Handler: mux,
	}
	log.Fatal(srv.ListenAndServe())

}