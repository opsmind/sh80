package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"
	"time"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("./sh80 <listen (ex: 0.0.0.0:80)> <template file path (ex: ./sh80.template)> <timeout_sec (ex: 30)>")
	}
	timeout, err := strconv.Atoi(os.Args[3])
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.ParseFiles(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	t = t.Funcs(template.FuncMap{
		"f_json": f_json, "f_split": f_split, "f_fields": f_fields, "f_debug": f_debug,
	})
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		log.Println("[request]: ", req)
		body := map[string]interface{}{}
		switch ct := req.Header.Get("Content-Type"); {
		case strings.HasPrefix(ct, "application/json"):
			buf, err := ioutil.ReadAll(req.Body)
			if err != nil {
				log.Println("[err]: ", err)
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
			}
			json.Unmarshal(buf, &body)
		default:
			log.Println("[warn]: unsupported content type", ct)
			w.WriteHeader(400)
			w.Write([]byte("unsupported content type"))
			return
		}

		var buf bytes.Buffer
		err = t.Execute(&buf, body)
		if err != nil {
			log.Println("[err]: ", err)
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
		}
		str := buf.String()
		log.Println("[debug]: ", str)
		cmds := strings.Split(str, "\n")
		for _, x := range cmds {
			ctx, cf := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
			defer cf()
			cmd := exec.CommandContext(ctx, "sh", "-c", x)
			stdout, err := cmd.Output()
			log.Println("[debug]: ", string(stdout))
			w.Write(stdout)
			if err != nil {
				w.Write([]byte("\n" + err.Error()))
			}
		}
	})
	log.Fatal(http.ListenAndServe(os.Args[1], nil))
}

var f_json = func(s string) (ret interface{}) {
	err := json.Unmarshal([]byte(s), &ret)
	if err != nil {
		return map[string]error{"opsmind_f_json_err:": err}
	}
	return
}

var f_split = func(sep, s string) (ret interface{}) {
	ret = strings.Split(s, sep)
	return
}

var f_fields = func(s string) (ret interface{}) {
	ret = strings.Fields(s)
	return
}

var f_debug = func(prefix string, s interface{}) (ret interface{}) {
	log.Println("sh80 debug[", prefix, "]: ", s)
	ret = s
	return
}
