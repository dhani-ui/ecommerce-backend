package main

import (
	"database/sql"
	"log"
	"net/http"
	"os" // Tambahkan os untuk membaca environment variable

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Barang struct {
	ID            int    `json:"id"`
	NamaBarang    string `json:"nama_barang"`
	Kuantiti      int    `json:"kuantiti"`
	Harga         int    `json:"harga"` // Tambahan Baru
	TanggalMasuk  string `json:"tanggal_masuk"`
	TanggalKeluar string `json:"tanggal_keluar"`
	GambarBarang  string `json:"gambar_barang"`
}


func main() {
	// 1. Ambil URL Database dari Environment Variable (disiapkan oleh hosting nanti)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Fallback ke lokal jika dijalankan di komputer sendiri
		dbURL = "user=postgres password=password_kamu dbname=ecommerce_db sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := gin.Default()
r.Use(cors.Default())

// Tambahkan baris ini agar folder uploads bisa diakses dari web
r.Static("/uploads", "./uploads")


	// Endpoint untuk mengambil semua barang
	// Endpoint untuk mengambil semua barang
	r.GET("/api/barang", func(c *gin.Context) {
		// 1. Mengambil data dari database
		rows, err := db.Query("SELECT id, nama_barang, kuantiti, COALESCE(harga, 0), COALESCE(CAST(tanggal_masuk AS VARCHAR), ''), COALESCE(CAST(tanggal_keluar AS VARCHAR), ''), COALESCE(gambar_barang, '') FROM barang")
		
		// Pengecekan error (Ini akan memperbaiki error "declared and not used: err")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		// 2. Mendeklarasikan variabel listBarang (Ini akan memperbaiki error "undefined: listBarang")
		var listBarang []Barang 

		// 3. Memasukkan data dari database ke dalam listBarang
		for rows.Next() {
			var b Barang
			if err := rows.Scan(&b.ID, &b.NamaBarang, &b.Kuantiti, &b.Harga, &b.TanggalMasuk, &b.TanggalKeluar, &b.GambarBarang); err != nil {
				continue
			}
			listBarang = append(listBarang, b)
		}

		// 4. Mencegah nilai "null" jika database masih kosong
		if listBarang == nil {
			listBarang = []Barang{}
		}

		// 5. Mengirimkan data ke React
		c.JSON(http.StatusOK, listBarang)
	})


	// Endpoint untuk MENAMBAH barang baru (Mendukung Upload File)
	r.POST("/api/barang", func(c *gin.Context) {
namaBarang := c.PostForm("nama_barang")
kuantiti := c.PostForm("kuantiti")
harga := c.PostForm("harga") // Ambil input harga
		tanggalMasuk := c.PostForm("tanggal_masuk")
		tanggalKeluar := c.PostForm("tanggal_keluar")

		// Proses Upload Gambar
		file, err := c.FormFile("gambar_barang")
		var filePath string

		if err == nil {
			// Jika ada file yang diupload, simpan ke folder uploads/
			filePath = "/uploads/" + file.Filename
			if err := c.SaveUploadedFile(file, "."+filePath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan gambar"})
				return
			}
		}

		// Query INSERT ke PostgreSQL
query := `
    INSERT INTO barang (nama_barang, kuantiti, harga, tanggal_masuk, tanggal_keluar, gambar_barang) 
    VALUES ($1, $2, $3, NULLIF($4, '')::DATE, NULLIF($5, '')::DATE, $6) 
    RETURNING id
`
		
		var id int
errDB := db.QueryRow(query, namaBarang, kuantiti, harga, tanggalMasuk, tanggalKeluar, filePath).Scan(&id)
		if errDB != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan ke database: " + errDB.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Barang & Gambar berhasil ditambahkan", "id": id})
	})


	// Endpoint untuk MENGEDIT/UPDATE barang (PUT)
	r.PUT("/api/barang/:id", func(c *gin.Context) {
		id := c.Param("id") // Ambil ID dari URL
		var b Barang
		
		if err := c.ShouldBindJSON(&b); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
			return
		}

		// ... di dalam r.PUT ...
query := `
    UPDATE barang 
    SET nama_barang = $1, kuantiti = $2, harga = $3, tanggal_masuk = NULLIF($4, '')::DATE, tanggal_keluar = NULLIF($5, '')::DATE, gambar_barang = $6 
    WHERE id = $7
`
_, err := db.Exec(query, b.NamaBarang, b.Kuantiti, b.Harga, b.TanggalMasuk, b.TanggalKeluar, b.GambarBarang, id)
// ...

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate data: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil diupdate"})
	})

	// Endpoint untuk MENGHAPUS barang (DELETE)
	r.DELETE("/api/barang/:id", func(c *gin.Context) {
		id := c.Param("id")

		_, err := db.Exec("DELETE FROM barang WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus data"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil dihapus"})
	})

	// 2. Ambil PORT dari Environment Variable (Render.com akan memberikan port dinamis)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	r.Run(":" + port)
}