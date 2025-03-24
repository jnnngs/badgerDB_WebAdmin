package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	badger "github.com/dgraph-io/badger/v4"
)

var db *badger.DB
var username = getEnv("BADGER_ADMIN_USER", "admin")
var password = getEnv("BADGER_ADMIN_PASS", "botfluence")

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func main() {
	opts := badger.DefaultOptions("./data/badger")
	opts.Logger = nil

	var err error
	db, err = badger.Open(opts)
	if err != nil {
		log.Fatalf("Failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/", withAuth(serveHTML))
	http.HandleFunc("/keys", withAuth(handleListKeys))
	http.HandleFunc("/get", withAuth(handleGetKey))
	http.HandleFunc("/set", withAuth(handleSetKey))
	http.HandleFunc("/delete", withAuth(handleDeleteKey))
	http.HandleFunc("/backup", withAuth(handleBackup))
	http.HandleFunc("/restore", withAuth(handleRestore))
	http.HandleFunc("/debug-count", withAuth(func(w http.ResponseWriter, r *http.Request) {
		count := 0
		db.View(func(txn *badger.Txn) error {
			it := txn.NewIterator(badger.DefaultIteratorOptions)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				count++
			}
			return nil
		})
		w.Write([]byte(fmt.Sprintf("Total keys: %d", count)))
	}))

	fmt.Println("BadgerAdmin UI running at http://localhost:8080 (user: admin)")
	http.ListenAndServe(":8080", nil)
}

func withAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			headerAuth(w)
			return
		}
		decoded, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 || parts[0] != username || parts[1] != password {
			headerAuth(w)
			return
		}
		handler(w, r)
	}
}

func headerAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>BadgerAdmin</title>
<style>
body { font-family: sans-serif; padding: 2rem; }
input, button, textarea { margin: 0.5rem; padding: 0.4rem; }
pre { background: #eee; padding: 1rem; max-height: 300px; overflow: auto; }
</style>
</head>
<body>
<h1>🛠️ BadgerAdmin</h1>
<div>
  <input id="prefix" placeholder="Prefix (optional)">
  <button onclick="loadKeys()">List Keys</button>
</div>
<ul id="keys"></ul>
<hr>
<div>
  <input id="getKey" placeholder="Key">
  <button onclick="getKey()">Get</button>
  <pre id="value"></pre>
</div>
<hr>
<div>
  <input id="setKey" placeholder="Key">
  <textarea id="setValue" rows="4" cols="50" placeholder="JSON value"></textarea><br>
  <button onclick="setKey()">Set</button>
</div>
<hr>
<div>
  <input id="delKey" placeholder="Key">
  <button onclick="deleteKey()">Delete</button>
</div>
<hr>
<div>
  <button onclick="window.location='/backup'">🔽 Download Backup</button>
  <form id="restoreForm">
    <input type="file" name="backup" id="restoreInput">
    <button type="submit">🔼 Upload & Restore</button>
  </form>
  <div id="restoreStatus"></div>
</div>
<hr>
<div>
  <button onclick="getKeyCount()">🔍 Show Total Key Count</button>
  <div id="keyCountStatus"></div>
</div>
<script>
function loadKeys() {
  const p = document.getElementById('prefix').value;
  fetch('/keys?prefix=' + encodeURIComponent(p), { headers: authHeader() })
    .then(res => res.json())
    .then(data => {
      const list = document.getElementById('keys');
      list.innerHTML = '';
      data.forEach(k => {
        const li = document.createElement('li');
        li.textContent = k;
        list.appendChild(li);
      });
    });
}

function getKey() {
  const key = document.getElementById('getKey').value;
  fetch('/get?key=' + encodeURIComponent(key), { headers: authHeader() })
    .then(res => res.json())
    .then(data => {
      document.getElementById('value').textContent = JSON.stringify(data, null, 2);
      document.getElementById('setKey').value = key;
      document.getElementById('setValue').value = JSON.stringify(data, null, 2);
    });
}

function setKey() {
  const key = document.getElementById('setKey').value;
  const value = document.getElementById('setValue').value;
  fetch('/set', {
    method: 'POST',
    headers: Object.assign(authHeader(), { 'Content-Type': 'application/json' }),
    body: JSON.stringify({ key, value: JSON.parse(value) })
  });
}

function deleteKey() {
  const key = document.getElementById('delKey').value;
  fetch('/delete?key=' + encodeURIComponent(key), {
    method: 'POST', headers: authHeader()
  });
}

function authHeader() {
  return { 'Authorization': 'Basic ' + btoa('admin:botfluence') };
}

function getKeyCount() {
  fetch('/debug-count', { headers: authHeader() })
    .then(res => res.text())
    .then(text => {
      document.getElementById('keyCountStatus').innerText = text;
    })
    .catch(err => {
      document.getElementById('keyCountStatus').innerText = '❌ Error fetching key count';
    });
}

document.getElementById('restoreForm').addEventListener('submit', async function(e) {
  e.preventDefault();
  const fileInput = document.getElementById('restoreInput');
  const formData = new FormData();
  formData.append('backup', fileInput.files[0]);
  const res = await fetch('/restore', {
    method: 'POST',
    body: formData,
    headers: authHeader()
  });
  const text = await res.text();
  document.getElementById('restoreStatus').innerText = res.ok ? '✅ ' + text : '❌ ' + text;
});
</script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleListKeys(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	var keys []string
	db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			key := it.Item().Key()
			if prefix == "" || strings.HasPrefix(string(key), prefix) {
				keys = append(keys, string(key))
			}
		}
		return nil
	})
	json.NewEncoder(w).Encode(keys)
}

func handleGetKey(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	var val []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			val = append([]byte{}, v...)
			return nil
		})
	})
	if err != nil {
		http.Error(w, "Key not found", 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(val)
}

func handleSetKey(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Key   string          `json:"key"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid input", 400)
		return
	}
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(payload.Key), payload.Value)
	})
	if err != nil {
		http.Error(w, "Failed to write key", 500)
		return
	}
	w.WriteHeader(200)
}

func handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
	if err != nil {
		http.Error(w, "Failed to delete key", 500)
		return
	}
	w.WriteHeader(200)
}

func handleBackup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=badger-backup.bak")
	w.Header().Set("Content-Type", "application/octet-stream")
	_, err := db.Backup(w, 0)
	if err != nil {
		http.Error(w, "Backup failed", 500)
		return
	}
}

func handleRestore(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20) // 10MB limit
	file, _, err := r.FormFile("backup")
	if err != nil {
		http.Error(w, "Failed to read backup file", 400)
		return
	}
	defer file.Close()

	err = db.Load(file, 10)
	if err != nil {
		http.Error(w, "Restore failed", 500)
		return
	}
	w.Write([]byte("Restore completed"))
}
