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

	// Endpoint untuk mengambil semua barang
	r.GET("/api/barang", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, nama_barang, kuantiti, COALESCE(CAST(tanggal_masuk AS VARCHAR), ''), COALESCE(CAST(tanggal_keluar AS VARCHAR), ''), COALESCE(gambar_barang, '') FROM barang")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var listBarang []Barang
		for rows.Next() {
			var b Barang
			if err := rows.Scan(&b.ID, &b.NamaBarang, &b.Kuantiti, &b.TanggalMasuk, &b.TanggalKeluar, &b.GambarBarang); err != nil {
				continue
			}
			listBarang = append(listBarang, b)
		}

		c.JSON(http.StatusOK, listBarang)
	})

	// Endpoint untuk MENAMBAH barang baru
	r.POST("/api/barang", func(c *gin.Context) {
		var b Barang
		
		// Bind JSON dari request body ke struct Barang
		if err := c.ShouldBindJSON(&b); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid: " + err.Error()})
			return
		}

		// Query INSERT ke PostgreSQL
		// Kita menggunakan NULLIF untuk menangani jika frontend mengirim string kosong pada tanggal
		query := `
			INSERT INTO barang (nama_barang, kuantiti, tanggal_masuk, tanggal_keluar, gambar_barang) 
			VALUES ($1, $2, NULLIF($3, '')::DATE, NULLIF($4, '')::DATE, $5) 
			RETURNING id
		`
		
		var id int
		err := db.QueryRow(query, b.NamaBarang, b.Kuantiti, b.TanggalMasuk, b.TanggalKeluar, b.GambarBarang).Scan(&id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan ke database: " + err.Error()})
			return
		}

		// Kembalikan response sukses beserta ID barang yang baru dibuat
		b.ID = id
		c.JSON(http.StatusCreated, gin.H{
			"message": "Barang berhasil ditambahkan",
			"data": b,
		})
	})

	// Endpoint untuk MENGEDIT/UPDATE barang (PUT)
	r.PUT("/api/barang/:id", func(c *gin.Context) {
		id := c.Param("id") // Ambil ID dari URL
		var b Barang
		
		if err := c.ShouldBindJSON(&b); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
			return
		}

		query := `
			UPDATE barang 
			SET nama_barang = $1, kuantiti = $2, tanggal_masuk = NULLIF($3, '')::DATE, tanggal_keluar = NULLIF($4, '')::DATE, gambar_barang = $5 
			WHERE id = $6
		`
		_, err := db.Exec(query, b.NamaBarang, b.Kuantiti, b.TanggalMasuk, b.TanggalKeluar, b.GambarBarang, id)
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