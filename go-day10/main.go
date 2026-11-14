package main
import (
	"net/http"
	"log"
)

func main(){
	store:=NewStore()
	handler:=NewHandler(store)
	mux:=http.NewServeMux()
	mux.HandleFunc("POST /urls",handler.CreateUrlHandler)
	mux.HandleFunc("GET /urls",handler.ListURLsHandler)
	mux.HandleFunc("GET /urls/{code}",handler.GetURLHandler)
	mux.HandleFunc("DELETE /urls/{code}",handler.DeleteURLHandler)
	err:=http.ListenAndServe(":8080",mux)
	if err!=nil{
		log.Fatal(err)
	}

}