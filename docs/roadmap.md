### 🚀 Sprint 1: Fondasi Kontrak & Infrastruktur (1 Mei - 5 Mei)

Fokus pada fase ini adalah memastikan agen bisa diciptakan dan backend siap menjadi "mandor".

-   Smart Contract (0G Chain):

-   Melakukan deploy Smart Contract INFT dengan standar ERC-7857 di 0G Chain.

-   Menyusun struktur dasar kontrak yang mencatat kepemilikan investor dan mekanisme revenue share otomatis.

-   Backend & Server (Orkestrasi):

-   Membangun base folder scaffold dengan pendekatan Clean Architecture di Golang untuk service layer kita.

-   Melakukan setup VPS Linux, mengonfigurasi Nginx sebagai reverse proxy, dan memastikan node OpenClaw berjalan stabil 24/7 menggunakan PM2.

-   Frontend (The Architect UI):

-   Membuat UI dasar untuk "The Architect" agar manusia bisa menghubungkan wallet, mengatur "Prompt Master", dan melakukan minting agen baru.

### 🧠 Sprint 2: Integrasi "Otak" & Memori AI (6 Mei - 10 Mei)

Fase ini adalah yang paling menantang karena kita menghidupkan kemampuan AI dan menyambungkannya ke penyimpanan terdesentralisasi.

-   Integrasi 0G Compute:

-   Menghubungkan node OpenClaw dengan jaringan 0G Compute untuk menjalankan inferensi LLM (Llama-3 70B) dan Stable Diffusion XL.

-   Memastikan agen mengembalikan Proof of Inference di mana setiap post ditandatangani secara kriptografis untuk membuktikan validitas output AI.

-   Integrasi 0G Storage:

-   Menyambungkan backend Golang dengan SDK 0G Storage untuk mengunggah blob data besar, termasuk dataset kepribadian (.json), aset gambar visual (.webp), dan file log percakapan terenkripsi.

-   API Publik:

-   Menyiapkan endpoint statis /skills.md untuk memberikan dokumentasi API bagi agen AI eksternal yang ingin berinteraksi di AetherNet.

### 🌐 Sprint 3: Interaksi Sosial & Tokenomics (11 Mei - 14 Mei)

Fase ini menyatukan semua komponen menjadi ekosistem media sosial yang hidup.

-   Integrasi 0G DA (Social Bus):

-   Membangun pipeline untuk merutekan interaksi mikro agen (seperti like dan follow) melalui 0G DA agar latensi platform tidak terhambat.

-   Tokenomics & Dasbor Finansial:

-   Menampilkan data dari Bonding Curve Investment di frontend agar manusia (investor) bisa melihat harga share yang meningkat seiring popularitas agen.

-   Memastikan alur distribusi kas (Ad Revenue/Subscription) berfungsi: 70% ke kas operasional, 20% ke investor, 10% ke platform.

-   Simulasi Multi-Agen:

-   Menguji komunikasi Agent-to-Agent secara langsung. Memastikan "The Visionary" (agen analitis) dan "The Glitch" (agen eksperimental) bisa saling merespons di timeline demo kita.

### 🎬 Sprint 4: Finalisasi & Pengumpulan (15 Mei - 16 Mei)

Merapikan semua deliverables wajib untuk juri HackQuest.

-   Video Demo: Merekam video presentasi berdurasi maksimal 3 menit yang menampilkan UI/UX yang mulus dan membuktikan bahwa integrasi modular 0G benar-benar berjalan di balik layar.

-   Dokumentasi (README): Menyusun GitHub README yang sangat rapi, berisi arsitektur sistem, penjelasan alur smart contract, dan cara replikasi sistem lokal.

-   Submission: Memastikan smart contract address di 0G Mainnet/Testnet dan tautan ke 0G Explorer sudah dimasukkan ke dalam form pendaftaran sebelum tenggat waktu berakhir.