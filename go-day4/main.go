package main

import (
	"fmt"
	"encoding/json"
	"io"
	"strings"
	"os"
	"bytes"
)

func writeAll(w io.Writer,name  string){// here io.Writer is an interface and we can use it to write to any destination that implements the Writer interface
	fmt.Fprintf(w,"hello %s\n",name)// here we are using fmt.Fprintf to write to the writer w and we are passing the name as a parameter


}

func readAll(r io.Reader) (int,error){// here io.Reader is an interface and we can use it to read from any source that implements the Reader interface
//(int,error) is the return type of the function and it returns the number of bytes read and an error if any
buf:=make([]byte ,4)
total:=0
for {
	n,err:=r.Read(buf)
	if n>0{
		fmt.Println("read",n,"bytes",string(buf[:n]))
		total+=n
	}
	if err==io.EOF{
		return total,nil
	}
	if err!=nil{
		return total,err
	}
}

}
func main(){
	type user struct{
		ID  int `json:"id"`
		Name string `json:"name"`
		Email string `json:"email,omitempty"`// here omitempty will omit the field if it is empty
		password string
	}
	u := user{ID:1, Name:"John",Email:"",password:"secret"}// here password is unexported field and will not be included in the JSON output
	u1:= user{ID:2,Name:"kailash",Email:"kailash1@gmail.com",password:"123"}
	res,err:= json.Marshal(u)
	re,e:=json.Marshal(u1)
	if err!=nil{
		fmt.Println(err)
	}else{
		fmt.Println(string(res))
	}
	if e!=nil{
		fmt.Println(e)
	}else{
		fmt.Println(string(re))
	}

	//unmarshalling
	data:=[]byte(`{"id":3,"name":"mahesh","email":"ako0@gmail.com","password":"hulk"}`)
	// in the above data password is unexported field and will not be included
	var u2 user
	err2:=json.Unmarshal(data,&u2)
	if err2!=nil{
		fmt.Println(err2)
	}else{
		fmt.Println(u2)
	}
	//reader-->think reader as a source of data that can be read from, like a file or a network connection. It is an interface that defines the Read method, which reads data from the source and returns the number of bytes read and an error if any.
	r:=strings.NewReader("hello world")
	val,er:=readAll(r)
	fmt.Println("total bytes read:",val)
	fmt.Println("error:",er)

	//writer--> think writer as a destination where data can be written to, like a file or a network connection. It is an interface that defines the Write method, which writes data to the destination and returns the number of bytes written and an error if any.
	//os.stdout is a writer that writes to the standard output (console)
	writeAll(os.Stdout,"kailash")
	//&bytes.buffer is a writer that writes to a buffer in memory
	var b bytes.Buffer
	writeAll(&b,"mahesh")
	fmt.Printf("buffer contents:%q\n",b.String())

	// encoder -> think encoder as a tool that converts data from one format to another, like converting a struct to JSON. It is an interface that defines the Encode method, which encodes data and writes it to the destination.
	var ak bytes.Buffer
	enc:=json.NewEncoder(&ak).Encode(u1)
	if enc!=nil{
		fmt.Println("error encoding JSON:", enc)
	}else{
		fmt.Printf("encoded JSON: %s\n", ak.String())
	}

	enc1:=json.NewEncoder(os.Stdout).Encode(u1)
	if enc1!=nil{
		fmt.Println("error encoding JSON:", enc1)
	}


	
}