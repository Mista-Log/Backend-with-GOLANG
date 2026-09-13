// Chat Server — run with: go mod tidy && go run .
// Then open http://localhost:8080/?name=ada in two different browser
// tabs (or windows) and chat between them.
package main

import (
	"fmt"
	"log"
	"net/http"

	"chatserver/internal/hub"
	"chatserver/internal/server"
)

const clientPage = `<!DOCTYPE html>
<html>
<body>
	<div id="messages" style="height:300px;overflow-y:scroll;border:1px solid #ccc;padding:8px;"></div>
	<input id="input" style="width:80%" placeholder="Type a message and press Enter">
	<script>
		const params = new URLSearchParams(window.location.search);
		const name = params.get("name") || "anonymous";
		const ws = new WebSocket("ws://" + window.location.host + "/ws?name=" + name);
		const messages = document.getElementById("messages");
		ws.onmessage = (event) => {
			const div = document.createElement("div");
			div.textContent = event.data;
			messages.appendChild(div);
			messages.scrollTop = messages.scrollHeight;
		};
		document.getElementById("input").addEventListener("keydown", (e) => {
			if (e.key === "Enter" && e.target.value) {
				ws.send(e.target.value);
				e.target.value = "";
			}
		});
	</script>
</body>
</html>`

func main() {
	h := hub.New()
	go h.Run() // the ONE Hub goroutine — see internal/hub's package comment

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, clientPage)
	})
	mux.HandleFunc("GET /ws", server.NewHandler(h))

	fmt.Println("Chat server listening on http://localhost:8080")
	fmt.Println("Open http://localhost:8080/?name=ada and http://localhost:8080/?name=bob in two tabs")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
