package halgo

func Example() {
	if server, err := NewServer(RootResource{}); err != nil {
		Print(err)
	} else {
		Print(server)
	}
	// Output:
	// not this!
}
