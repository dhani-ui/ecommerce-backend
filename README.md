created using go languange

sql command :
CREATE DATABASE ecommerce_db;
\c ecommerce_db;
CREATE TABLE barang (
    id SERIAL PRIMARY KEY,
    nama_barang VARCHAR(255) NOT NULL,
    kuantiti INT NOT NULL DEFAULT 0,
    tanggal_masuk DATE,
    tanggal_keluar DATE,
    gambar_barang TEXT -- Menyimpan URL atau path gambar
);

-- Masukkan data dummy untuk testing
INSERT INTO barang (nama_barang, kuantiti, tanggal_masuk, gambar_barang) 
VALUES ('Laptop Gaming', 10, '2026-03-16', 'https://via.placeholder.com/150');
