package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

func main() {
	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		cmd := exec.Command("./cli/speedtest", "--accept-license", "--accept-gdpr", "--format=json")
		output, err := cmd.Output()
		if err != nil {
			http.Error(
				w,
				fmt.Sprintf(`{"error": "Failed to run speedtest: %s"}`, err.Error()),
				http.StatusInternalServerError,
			)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(output)
	})

	fmt.Println("Server started at http://localhost:8080")
	fmt.Println("Access /run endpoint to execute a speedtest")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
