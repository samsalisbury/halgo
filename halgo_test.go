package halgo

import (
	httpspec "github.com/opentable/http-specs"
	"net/http"
	"testing"
)

func Test_Example(t *testing.T) {
	if graph, err := Graph(RootResource{}); err != nil {
		Print(err)
	} else {
		Print("Listening on :8080")
		http.ListenAndServe(":8080", graph)
		if spec, err := httpspec.NewSpec("halgo.httpspec"); err != nil {
			t.Error(err)
		} else {
			spec.Run(t, "http://localhost:8080")
		}

	}
	// Output:
	// not this!
}

// func Test_Example(t *testing.T) {
// 	if server, err := NewServer(RootResource{}); err != nil {
// 		Print(err)
// 	} else {
// 		Print("Listening on :8080")
// 		go http.ListenAndServe(":8080", server)
// 		if spec, err := httpspec.NewSpec("halgo.httpspec"); err != nil {
// 			t.Error(err)
// 		} else {
// 			spec.Run(t, "http://localhost:8080")
// 		}

// 	}
// 	// Output:
// 	// not this!
// }
