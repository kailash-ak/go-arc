package main

// net package
import (
	"fmt"
	"net"
	"time"
)

func main() {
	domain := "google.com"
	//LookupHost looks up the given host using the local resolver. It returns a slice of that host's addresses.
	//LookupHost uses context.Background internally; to specify the context, use [Resolver.LookupHost].
	for i := 0; i < 2; i++ {
		start := time.Now()
		ips, err := net.LookupHost(domain) //
		end := time.Since(start)
		if err != nil {
			fmt.Println("DNS lookup failed:", err)
			continue
		}
		fmt.Println("Attempt", i)
		fmt.Println(ips)
		fmt.Println(end.Milliseconds())
		//fmt.Println("elapsed time:",end-start)

	}
	// in the above for loop we have seen the lookup is reduced from attempt 0 to attempt 1

	//ok now we try to establish a tcp connection
	conn, err1 := net.Dial("tcp", "google.com:80")
	if err1 != nil {
		fmt.Println("tcp connection error:", err1)
	} else {
		fmt.Println(conn.LocalAddr())
	}
	defer conn.Close()
	dummy()
	//fmt.Println(conn)
	fmt.Println("Go day 6")
}
