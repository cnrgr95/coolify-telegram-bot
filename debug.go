package main

import (
    "context"
    "fmt"
    "io"
    "net"
    "net/http"
)

func handleDebugMongo(w http.ResponseWriter, r *http.Request) {
    client := http.Client{
        Transport: &http.Transport{
            DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
                return net.Dial("unix", "/var/run/docker.sock")
            },
        },
    }
    
    // Get containers to find mongodb-telegram-bot ID
    req, _ := http.NewRequest("GET", "http://localhost/containers/json?all=1", nil)
    resp, err := client.Do(req)
    if err != nil {
        w.Write([]byte(err.Error()))
        return
    }
    body, _ := io.ReadAll(resp.Body)
    resp.Body.Close()
    
    // Then get logs of mongodb-telegram-bot (hardcoding name usually works)
    reqLogs, _ := http.NewRequest("GET", "http://localhost/containers/mongodb-telegram-bot/logs?stdout=1&stderr=1&tail=50", nil)
    respLogs, err := client.Do(reqLogs)
    if err != nil {
        w.Write([]byte("Logs error: " + err.Error()))
        return
    }
    logsBody, _ := io.ReadAll(respLogs.Body)
    respLogs.Body.Close()
    
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte("Containers JSON:\n"))
    w.Write(body)
    w.Write([]byte("\n\nLogs:\n"))
    w.Write(logsBody)
}
