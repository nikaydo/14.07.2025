package handlers

import (
	"fmt"
	"main/internal/config"
	"main/internal/filework"
	"main/internal/task"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Handlers struct {
	Tasks *task.Task
	Env   config.Config
}

func Init(env *config.Config) {
	handlers := Handlers{Tasks: &task.Task{Mutex: &sync.Mutex{}, List: make(map[int]filework.Zip)}, Env: *env}
	http.HandleFunc("/api/zip", handlers.MakeTask)
	http.HandleFunc("/api/zip/add", handlers.AddToTask)
	http.HandleFunc("/api/zip/status", handlers.StatusTask)
}

func (h *Handlers) MakeTask(w http.ResponseWriter, r *http.Request) {
	id, err := h.Tasks.Add(h.Env.MAX_TASKS, h.Env)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	fmt.Fprintf(w, "%d", id)
}

func (h *Handlers) AddToTask(w http.ResponseWriter, r *http.Request) {
	rawid := r.FormValue("id")
	url := r.FormValue("url")
	id, err := strconv.Atoi(rawid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error: %v", err)
	}
	zip, ok := h.Tasks.List[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Task not found")
		return
	}

	str := strings.Split(url, ",")
	badUrl := ""
	for _, i := range str {
		if len(zip.Urls) >= h.Env.MAX_FILES_IN_ZIP {
			badUrl += i + " not added - archive is full. "
			continue
		}
		if _, err := filework.GetFileFromUrl(i, true); err != nil {
			badUrl += i + fmt.Sprintf(" file url not valid %v. ", err)
			continue
		}
		if err := zip.AppendUrl(i); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%v", err)
			continue
		}
	}

	if badUrl != "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", badUrl)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handlers) StatusTask(w http.ResponseWriter, r *http.Request) {
	rawid := r.FormValue("id")
	id, err := strconv.Atoi(rawid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}
	z, ok := h.Tasks.List[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Task not found")
		return
	}
	if len(z.Urls) <= h.Env.MAX_FILES_IN_ZIP-1 {
		s := z.Status
		fmt.Fprintf(w, *s)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=files.zip")
	file, err := z.MakeZip()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}
	_, err = w.Write(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}
	h.Tasks.Drop(id)
}
