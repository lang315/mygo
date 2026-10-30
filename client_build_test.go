//go:build !js

package mygo

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type clientBuildService struct{}

func (clientBuildService) Hello(name string) string { return name }

// TestWriteClientFollowsTheBuild asks for the client of an app that binds a
// service, or one that declares only an event, as MYGO_GENERATE does: a
// build with the generator writes it, and a production build
// (mygo_noinspector) says on stderr that it cannot and fails, rather than
// write nothing and leave mygo generate to think the app binds nothing.
func TestWriteClientFollowsTheBuild(t *testing.T) {
	for _, tc := range []struct {
		name    string
		declare func()
	}{
		{"Bind", func() {
			if err := bind("ClientBuildService", clientBuildService{}, false); err != nil {
				t.Fatal(err)
			}
			linkClient() // what Bind does
		}},
		{"NewEvent", func() { NewEvent[int]("test:client-build") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ipc.Lock()
			services, events, write := ipc.services, ipc.events, ipc.writeClient
			ipc.services, ipc.events, ipc.writeClient = nil, nil, nil
			ipc.Unlock()
			t.Cleanup(func() {
				ipc.Lock()
				ipc.services, ipc.events, ipc.writeClient = services, events, write
				ipc.Unlock()
			})
			tc.declare()

			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			stderr := os.Stderr
			os.Stderr = w
			out := filepath.Join(t.TempDir(), "src", "mygo.ts")
			werr := writeClient(out)
			os.Stderr = stderr
			w.Close()
			said, _ := io.ReadAll(r)
			_, statErr := os.Stat(out)

			if clientBuiltOut {
				if werr == nil {
					t.Error("a build without the generator asked for a client returned no error")
				}
				if !strings.Contains(string(said), "mygo_noinspector") {
					t.Errorf("a build without the generator said %q on stderr", said)
				}
				if statErr == nil {
					t.Error("a build without the generator wrote a client")
				}
				return
			}
			if werr != nil || len(said) != 0 {
				t.Fatalf("writeClient returned %v and said %q", werr, said)
			}
			if statErr != nil {
				t.Fatalf("the client was not written: %v", statErr)
			}
		})
	}
}

// TestProductionBuildSaysItLeavesTheGeneratorOut runs the test above in a
// build with mygo_noinspector.
func TestProductionBuildSaysItLeavesTheGeneratorOut(t *testing.T) {
	if clientBuiltOut {
		t.Skip("this build is the production build")
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go tool")
	}
	cmd := exec.Command(goTool, "test", "-tags", "mygo_noinspector", "-count=1", "-run", "^TestWriteClientFollowsTheBuild$", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the production build:\n%v\n%s", err, out)
	}
}
