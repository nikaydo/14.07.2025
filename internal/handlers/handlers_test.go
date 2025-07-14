package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"main/internal/config"
	"main/internal/filework"
	"main/internal/task"
)

func setupTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".txt") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "dummy content")
			return
		}
		if strings.HasSuffix(r.URL.Path, ".jpg") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "jpeg image data")
			return
		}
		http.NotFound(w, r)
	}))
}

func setupHandlers() (*Handlers, *config.Config) {
	env := config.Config{
		MAX_TASKS:          5,
		MAX_FILES_IN_ZIP:   2,
		ALLOWED_EXTENSIONS: []string{"txt", "jpg"},
	}
	handlers := Handlers{
		Tasks: &task.Task{
			Mutex: &sync.Mutex{},
			List:  make(map[int]filework.Zip),
		},
		Env: env,
	}
	return &handlers, &env
}

func TestMakeTaskHandler(t *testing.T) {
	handlers, _ := setupHandlers()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/zip", nil)
		rr := httptest.NewRecorder()

		handlers.MakeTask(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		body := rr.Body.String()
		if _, err := strconv.Atoi(body); err != nil {
			t.Errorf("handler returned unexpected body: got %v, expected an integer ID", body)
		}
	})

	t.Run("TaskLimitReached", func(t *testing.T) {
		for i := 0; i < handlers.Env.MAX_TASKS; i++ {
			handlers.Tasks.List[i] = *filework.PrepareZip(handlers.Env)
		}

		req := httptest.NewRequest("POST", "/api/zip", nil)
		rr := httptest.NewRecorder()

		handlers.MakeTask(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}

		expectedError := "server is busy"
		if !strings.Contains(rr.Body.String(), expectedError) {
			t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedError)
		}
	})
}
func TestAddToTaskHandler(t *testing.T) {
	handlers, env := setupHandlers()
	server := setupTestServer()
	defer server.Close()

	taskID := 1
	handlers.Tasks.List[taskID] = *filework.PrepareZip(*env)

	t.Run("Success", func(t *testing.T) {
		fileURL := server.URL + "/file1.txt"
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/zip/add?id=%d&url=%s", taskID, fileURL), nil)
		rr := httptest.NewRecorder()

		handlers.AddToTask(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}
	})

	t.Run("TaskNotFound", func(t *testing.T) {
		fileURL := server.URL + "/file1.txt"
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/zip/add?id=999&url=%s", fileURL), nil)
		rr := httptest.NewRecorder()

		handlers.AddToTask(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("FileLimitReached", func(t *testing.T) {
		limitTaskID := 2
		handlers.Tasks.List[limitTaskID] = *filework.PrepareZip(*env)
		for i := 0; i < env.MAX_FILES_IN_ZIP; i++ {
			zip := handlers.Tasks.List[limitTaskID]
			zip.AppendUrl(fmt.Sprintf(server.URL+"/%d.txt", i))
		}

		fileURL := server.URL + "/another.txt"
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/zip/add?id=%d&url=%s", limitTaskID, fileURL), nil)
		rr := httptest.NewRecorder()

		handlers.AddToTask(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("InvalidURL", func(t *testing.T) {
		invalidURL := "http://invalid-url"
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/zip/add?id=%d&url=%s", taskID, invalidURL), nil)
		rr := httptest.NewRecorder()

		handlers.AddToTask(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

func TestStatusTaskHandler(t *testing.T) {
	handlers, env := setupHandlers()
	server := setupTestServer()
	defer server.Close()

	t.Run("InProgressStatus", func(t *testing.T) {
		taskID := 1
		handlers.Tasks.List[taskID] = *filework.PrepareZip(*env)
		zip := handlers.Tasks.List[taskID]
		zip.AppendUrl(server.URL + "/file1.txt")

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/zip/status?id=%d", taskID), nil)
		rr := httptest.NewRecorder()

		handlers.StatusTask(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedStatus := "Prepare zip, u can add another 1"
		if rr.Body.String() != expectedStatus {
			t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedStatus)
		}
	})

	t.Run("DownloadZip", func(t *testing.T) {
		zipTaskID := 2
		handlers.Tasks.List[zipTaskID] = *filework.PrepareZip(*env)
		z := handlers.Tasks.List[zipTaskID]
		z.AppendUrl(server.URL + "/file1.txt")
		z = handlers.Tasks.List[zipTaskID]
		z.AppendUrl(server.URL + "/image.jpg")

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/zip/status?id=%d", zipTaskID), nil)
		rr := httptest.NewRecorder()

		handlers.StatusTask(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		if ctype := rr.Header().Get("Content-Type"); ctype != "application/zip" {
			t.Errorf("wrong Content-Type header: got %s want application/zip", ctype)
		}
		if cdisp := rr.Header().Get("Content-Disposition"); cdisp != "attachment; filename=files.zip" {
			t.Errorf("wrong Content-Disposition header: got %s want attachment; filename=files.zip", cdisp)
		}

		zipReader, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
		if err != nil {
			t.Fatalf("failed to read zip archive: %v", err)
		}

		if len(zipReader.File) != 2 {
			t.Errorf("expected 2 files in zip, got %d", len(zipReader.File))
		}

		if _, ok := handlers.Tasks.List[zipTaskID]; ok {
			t.Error("task should be dropped after downloading the zip file")
		}
	})

	t.Run("TaskNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/zip/status?id=999", nil)
		rr := httptest.NewRecorder()

		handlers.StatusTask(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}
