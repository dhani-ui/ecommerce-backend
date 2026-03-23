created using go languange

psql :
CREATE DATABASE ecommerce_db;
\c ecommerce_db;

CREATE USER postgres;

CREATE TABLE barang (
    id SERIAL PRIMARY KEY,
    nama_barang VARCHAR(255) NOT NULL,
    kuantiti INT NOT NULL DEFAULT 0,
    tanggal_masuk DATE,
    tanggal_keluar DATE,
  harga BIGINT DEFAULT 0,
    gambar_barang TEXT -- Menyimpan URL atau path gambar
);


-- Memberikan izin untuk Read, Insert, Update, dan Delete di tabel barang
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE barang TO postgres;

-- Memberikan izin pada "sequence" agar ID bisa bertambah otomatis (auto-increment) saat menambah barang baru
GRANT USAGE, SELECT ON SEQUENCE barang_id_seq TO postgres;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;


CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password TEXT NOT NULL
);



type go main.go to run

fitur fitur :

guest :

add item to chart

checkout

form checkout


user / admin :

login jwt

add item add pic

edit item yang ada

add to chart / keranjang

form checkout
