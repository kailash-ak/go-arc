package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"net/url"
	"errors"
	"fmt"
	"mime"
	"log"
)

type Handler struct{
	store *Store
}

func NewHandler(store *Store) *Handler{
	return &Handler{
		store: store,
	}
}

// first we build our first endpoint 
//post /urls
//Our handler needs to perform four operations:
//1. Read the request body.
//2. Convert JSON into a Go struct.
//3. Call h.store.Create().
//4. Return a JSON response with HTTP status 201 Created.

type CreateUrlRequest struct{
	LongUrl string `json:"longUrl"`
}
type CreateURLResponse struct {
    Code    string `json:"code"`
    LongURL string `json:"longUrl"`
}
type ErrorResponse struct{
	Error string `json:"error"`
}

func (h *Handler) CreateUrlHandler(w http.ResponseWriter, r *http.Request){
	var req CreateUrlRequest
	contentType:=r.Header.Get("Content-Type")
	mediaType,_,err2:=mime.ParseMediaType(contentType)// here _= params
	
	if err2!=nil || mediaType!="application/json"{
		writeError(w,http.StatusUnsupportedMediaType,"unsupported media type ")
		return
	}
	//if contentType!="application/json" {// here clients sometimes send like this Content-Type: application/json; charset=utf-8
		//so ur logic fails(commented) so for this use mime library
		//http.Error(w,"UnSupported Media type",StatusUnsupportedMediaType)
		//return
	//}
	maxUploadSize:=int64(1<<20) // which interprets it as int 
	r.Body=http.MaxBytesReader(w,r.Body,maxUploadSize) // but maxBytesReader expects int64 so cast it

	err:=json.NewDecoder(r.Body).Decode(&req)
	if err!=nil{
		var maxByteError *http.MaxBytesError
		if errors.As(err,&maxByteError){
			writeError(w,http.StatusRequestEntityTooLarge,"request body exceeds 1 mb limit ")
			return
		}
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err1:= validateURL(req.LongUrl); err1!=nil{
		writeError(w,http.StatusBadRequest,err1.Error())
		return
	}
	shortCode,e:=h.store.Create(req.LongUrl)
	if e!=nil{
		writeError(w,http.StatusInternalServerError,"Error at store")
		return
	}
	
	//w.Header().Set("Content-Type","application/json")
	//w.WriteHeader(http.StatusCreated)
	response:=CreateURLResponse{
		Code:shortCode,
		LongURL:req.LongUrl,
	}
	w.Header().Set("Location","/urls/"+shortCode)
	//json.NewEncoder(w).Encode(response)
	writeJson(w,http.StatusCreated,response)
}

func (h *Handler) GetURLHandler(w http.ResponseWriter, r *http.Request){
	//Extract the short code using r.PathValue("code").
	// Call h.store.Get(code).
	// Return 404 if the code doesn't exist.
	// Otherwise return the original URL as JSON with status 200 OK.
	shortcode:=r.PathValue("code")
	url,flag:=h.store.Get(shortcode)
	cleaned:=strings.TrimSpace(url)
	if cleaned=="" || flag== false{
		w.WriteHeader(http.StatusNotFound)
	}else{
		response:=CreateURLResponse{
			Code:shortcode,
			LongURL:url,
		}
		writeJson(w,http.StatusOK,response)
		//w.Header().Set("Content-Type","application/json")
		//w.WriteHeader(http.StatusOK)
		//json.NewEncoder(w).Encode(response)

	}

}

func (h *Handler)ListURLsHandler(w http.ResponseWriter, r *http.Request){
	// 	Call h.store.List().
	// Set Content-Type to application/json.
	// Return HTTP 200 OK.
	// Encode the returned slice using json.NewEncoder(w).
	urls:=h.store.List()
	//w.Header().Set("Content-Type","application/json")
	//w.WriteHeader(http.StatusOK)
	//json.NewEncoder(w).Encode(urls)
	writeJson(w,http.StatusOK,urls)
}

func(h *Handler) DeleteURLHandler(w http.ResponseWriter, r *http.Request){
	// 	Extract the code using r.PathValue("code").
	// Call h.store.Delete(code).
	// If it returns false, respond with 404 Not Found.
	// If it returns true, respond with 204 No Content.
	code:=r.PathValue("code")
	flag:=h.store.Delete(code)
	if flag==false{
		w.WriteHeader(http.StatusNotFound)
	}else{
		w.WriteHeader(http.StatusNoContent)
	}
}

func validateURL(input string) error{
	cleanedUrl:=strings.TrimSpace(input)
	if  cleanedUrl==""{
		return errors.New("invalid url")// here errors.New() creates a value u need to return that value
		
	}
	if len(cleanedUrl)>2048{
		return errors.New("url length must be <2048")
		
	}
	parsedUrl,err:=url.Parse(cleanedUrl)
	if err!=nil{
		return fmt.Errorf("url parsing failed: %w",err)
		
	}
	if strings.ToLower(parsedUrl.Scheme) != "https" && strings.ToLower(parsedUrl.Scheme) !="http"{
		return errors.New("invalid scheme")
	}
	if strings.TrimSpace(parsedUrl.Hostname())==""{
		
		return errors.New("invalid hostname")
	}
	return nil
}
func writeError(w http.ResponseWriter, status int, message string){
	// 	Set Content-Type to application/json.
	// Write the supplied HTTP status.
	// Create an ErrorResponse.
	// Encode it into the response.
	//here refactor the code to use writejson for the below ones
	// w.Header().Set("Content-Type","application/json")
	// w.WriteHeader(status)
	errResponse:=ErrorResponse{
		Error:message,
	}
	writeJson(w,status,errResponse)
	// json.NewEncoder(w).Encode(errResponse)
}

func writeJson(w http.ResponseWriter, status int, v any){
	// Set Content-Type
    // Write HTTP status
    // Encode v into the response
    // Handle encoding/writing error
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(status)
	err:=json.NewEncoder(w).Encode(v)
	if err!=nil{
		log.Printf("failed to encode the json: %v",err)
		
	}
}