package main
import "crypto/rand"

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"


func  generateCode() (string,error){
	randomBytes:=make([]byte, 7)
	var b [1]byte
	for i:=0;i<7;{
		_,err:=rand.Read(b[:])
		if err!=nil{
			return "", err
		}
		if b[0]>247{
			continue
		}
		randomBytes[i]=base62Chars[b[0]%62]
		i++
	}
	return string(randomBytes), nil
}