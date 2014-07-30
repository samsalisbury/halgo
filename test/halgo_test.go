package test

func Test_Halgo() {
	if server, err := NewServer(RootResource{}); err != nil {
		Print(err)
	} else {
		Print("Listening on :8080")
		http.ListenAndServe(":8080", server)
	}
}
