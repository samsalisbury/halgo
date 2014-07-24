package main

func main() {
	if server, err := NewServer(RootResource{}); err != nil {
		Print(err)
	} else {
		Print(server)
	}
}
