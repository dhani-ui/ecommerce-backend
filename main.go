package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/lib/pq"
)

// Struct untuk tabel Barang
type Barang struct {
	ID            int    `json:"id"`
	NamaBarang    string `json:"nama_barang"`
	Kuantiti      int    `json:"kuantiti"`
	Harga         int    `json:"harga"`
	TanggalMasuk  string `json:"tanggal_masuk"`
	TanggalKeluar string `json:"tanggal_keluar"`
	GambarBarang  string `json:"gambar_barang"`
}

// Struct untuk tabel User (Admin)
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Secret Key untuk JWT
var jwtSecret = []byte("rahasia_super_aman_123") // Kamu bisa ganti teks ini nantinya

func main() {
	// Setup koneksi Database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Ganti password_kamu dengan password PostgreSQL milikmu
		dbURL = "user=postgres password=password_kamu dbname=ecommerce_db sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Gagal koneksi ke database: ", err)
	}
	defer db.Close()

	r := gin.Default()

	// Setup CORS (Mengizinkan React mengakses API ini)
	r.Use(cors.Default())

	// Folder uploads agar gambar bisa diakses publik
	r.Static("/uploads", "./uploads")

	// =======================================================
	// 1. ENDPOINT AUTH (REGISTER & LOGIN)
	// =======================================================
	r.POST("/api/register", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
			return
		}

		// Hash (Enkripsi) Password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengenkripsi password"})
			return
		}

		// Simpan ke Database
		_, err = db.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", user.Username, string(hashedPassword))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Username mungkin sudah dipakai"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Admin berhasil didaftarkan"})
	})

	r.POST("/api/login", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
			return
		}

		// Ambil password yang di-hash dari database
		var hashedPassword string
		err := db.QueryRow("SELECT password FROM users WHERE username = $1", user.Username).Scan(&hashedPassword)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Username tidak ditemukan"})
			return
		}

		// Cocokkan password input dengan password di database
		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Password salah"})
			return
		}

		// Buat Token JWT (Berlaku 24 Jam)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": user.Username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})

		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	})

	// =======================================================
	// 2. MIDDLEWARE UNTUK MELINDUNGI RUTE KHUSUS ADMIN
	// =======================================================
	authMiddleware := func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Anda belum login (Token tidak ditemukan)"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid atau sudah kadaluarsa"})
			c.Abort()
			return
		}
		c.Next()
	}

	// =======================================================
	// 3. ENDPOINT BARANG (CRUD)
	// =======================================================

	// [READ] - PUBLIC: Pembeli bisa melihat barang (Tanpa authMiddleware)
	r.GET("/api/barang", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, nama_barang, kuantiti, COALESCE(harga, 0), COALESCE(CAST(tanggal_masuk AS VARCHAR), ''), COALESCE(CAST(tanggal_keluar AS VARCHAR), ''), COALESCE(gambar_barang, '') FROM barang")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var listBarang []Barang
		for rows.Next() {
			var b Barang
			if err := rows.Scan(&b.ID, &b.NamaBarang, &b.Kuantiti, &b.Harga, &b.TanggalMasuk, &b.TanggalKeluar, &b.GambarBarang); err != nil {
				continue
			}
			listBarang = append(listBarang, b)
		}

		if listBarang == nil {
			listBarang = []Barang{}
		}

		c.JSON(http.StatusOK, listBarang)
	})

	// [CREATE] - PROTECTED: Hanya Admin yang bisa menambah barang
	r.POST("/api/barang", authMiddleware, func(c *gin.Context) {
		namaBarang := c.PostForm("nama_barang")
		kuantiti := c.PostForm("kuantiti")
		harga := c.PostForm("harga")
		tanggalMasuk := c.PostForm("tanggal_masuk")
		tanggalKeluar := c.PostForm("tanggal_keluar")

		// Proses Upload Gambar
		file, err := c.FormFile("gambar_barang")
		var filePath string

		if err == nil {
			filePath = "/uploads/" + file.Filename
			if err := c.SaveUploadedFile(file, "."+filePath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan gambar"})
				return
			}
		}

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

	// [UPDATE] - PROTECTED: Hanya Admin yang bisa mengedit barang
	r.PUT("/api/barang/:id", authMiddleware, func(c *gin.Context) {
		id := c.Param("id")
		var b Barang
		
		if err := c.ShouldBindJSON(&b); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
			return
		}

		query := `
			UPDATE barang 
			SET nama_barang = $1, kuantiti = $2, harga = $3, tanggal_masuk = NULLIF($4, '')::DATE, tanggal_keluar = NULLIF($5, '')::DATE, gambar_barang = $6 
			WHERE id = $7
		`
		_, err := db.Exec(query, b.NamaBarang, b.Kuantiti, b.Harga, b.TanggalMasuk, b.TanggalKeluar, b.GambarBarang, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate data: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil diupdate"})
	})

	// [DELETE] - PROTECTED: Hanya Admin yang bisa menghapus barang
	r.DELETE("/api/barang/:id", authMiddleware, func(c *gin.Context) {
		id := c.Param("id")

		_, err := db.Exec("DELETE FROM barang WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus data"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil dihapus"})
	})

	// =======================================================
	// JALANKAN SERVER GOLANG
	// =======================================================
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Println("Server berjalan di port " + port)
	r.Run(":" + port)
}
