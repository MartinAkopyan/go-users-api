// Точка входа сервиса. Здесь нет бизнес-логики: main только собирает части
// (конфиг, логгер, база, миграции, сервер), запускает их и аккуратно останавливает.
// Аналог в Laravel: public/index.php + bootstrap/app.php.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MartinAkopyan/go-users-api/internal/config"
	"github.com/MartinAkopyan/go-users-api/migrations"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// main только вызывает run и решает, с каким кодом выйти.
// Вся работа вынесена в run, потому что os.Exit (и log.Fatal, который его вызывает)
// завершает процесс сразу и НЕ выполняет defer. А в run все выходы идут через
// return, поэтому defer (закрытие пула, db) срабатывают всегда.
func main() {
	err := run()

	if err != nil {
		// slog.Error без своего логгера пишет через логгер по умолчанию:
		// наш logger создан внутри run и здесь недоступен.
		slog.Error("service stopped with error", "err", err)
		// Ненулевой код выхода говорит Docker/CI/оболочке: "программа упала".
		os.Exit(1)
	}
}

// run поднимает сервис по шагам и работает, пока не придёт сигнал остановки
// или не упадёт сервер. Любая ошибка возвращается наверх в main.
func run() error {
	// ---- Шаг 1: конфиг ----
	// Читаем переменные окружения (DATABASE_URL, HTTP_ADDR). Если чего-то
	// обязательного нет, падаем сразу с понятной ошибкой, а не где-то позже.
	cfg, err := config.Load()

	if err != nil {
		// %w "заворачивает" исходную ошибку внутрь новой: в логе видно и шаг
		// ("load config"), и причину. Позже её можно достать через errors.Is/As.
		return fmt.Errorf("load config: %w", err)
	}

	// ---- Шаг 2: логгер ----
	// slog пишет структурированные логи: сообщение + пары ключ-значение.
	// JSONHandler выводит каждую запись строкой JSON в stdout, откуда её
	// забирает Docker (docker compose logs app).
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Канал, в который Go положит сигнал ОС, когда процесс попросят остановиться:
	// os.Interrupt это Ctrl+C, SIGTERM присылает docker compose stop.
	// Буфер 1, чтобы сигнал не потерялся, если в момент прихода его никто не ждёт.
	// Подписываемся заранее, чтобы ловить сигнал на любом этапе старта.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// ---- Шаг 3: пул соединений с базой ----
	// Пул держит несколько открытых соединений и раздаёт их запросам, чтобы не
	// подключаться заново на каждый запрос. pgxpool.New только разбирает строку
	// подключения и создаёт пул, реальное подключение происходит лениво.
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)

	if err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}
	// defer ставим сразу после успешного создания: пул закроется при ЛЮБОМ
	// выходе из run (и при ошибке ниже, и при штатной остановке).
	defer pool.Close()

	// Ping реально идёт в базу. Здесь всплывут неверный пароль, недоступный хост
	// и т.п. Без него сервис "стартовал бы", а упал бы на первом запросе.
	err = pool.Ping(context.Background())

	if err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}

	logger.Info("DB connected")

	// ---- Шаг 4: миграции ----
	// goose работает со стандартным *sql.DB (database/sql), а у нас *pgxpool.Pool.
	// OpenDBFromPool это адаптер: *sql.DB, который берёт соединения из нашего пула.
	// db.Close закрывает только обёртку, сам пул продолжает работать.
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	// Provider это объект goose, который знает: какая база (диалект Postgres),
	// через что к ней ходить (db) и где лежат файлы миграций (migrations.FS,
	// вшитые в бинарник через //go:embed в migrations/migrations.go).
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)

	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	// Up применяет только те миграции, которых ещё нет в таблице goose_db_version.
	// Возвращает слайс результатов: по одному на каждую миграцию, применённую
	// сейчас. При повторном старте слайс пустой.
	result, err := provider.Up(context.Background())

	if err != nil {
		// Работать с неполной схемой нельзя, поэтому останавливаем сервис.
		return fmt.Errorf("could not execute up: %w", err)
	}

	// Сам goose в наш slog не пишет, поэтому логируем применённые миграции сами.
	// Duration в JSON выводится числом в наносекундах.
	for _, res := range result {
		logger.Info("migration applied", "version", res.Source.Version, "duration", res.Duration)
	}

	// ---- Шаг 5: роутер и HTTP-сервер ----
	// Роутер сопоставляет метод + путь с хендлером (как routes/api.php в Laravel).
	// Маршрутов пока нет, поэтому на любой запрос ответ 404.
	r := chi.NewRouter()

	// & берёт адрес структуры: srv это указатель *http.Server. Методы сервера
	// (ListenAndServe, Shutdown) работают с одним и тем же объектом.
	srv := &http.Server{
		Addr:    cfg.HTTPAddr, // где слушать, формат "хост:порт", ":8080" = все интерфейсы
		Handler: r,            // кто обрабатывает запросы
		// Таймауты защищают от клиентов, которые держат соединение бесконечно.
		ReadHeaderTimeout: 5 * time.Second,  // сколько ждать заголовки (защита от slowloris)
		ReadTimeout:       10 * time.Second, // сколько ждать весь запрос вместе с телом
		WriteTimeout:      10 * time.Second, // сколько времени есть на отправку ответа
		IdleTimeout:       60 * time.Second, // сколько держать keep-alive соединение без запросов
	}

	logger.Info("starting server", "addr", cfg.HTTPAddr)

	// ---- Шаг 6: запуск сервера ----
	// Канал для ошибки сервера. Горутина не может вернуть значение в run через
	// return, поэтому передаёт ошибку через канал. Буфер 1: горутина положит
	// ошибку и завершится, даже если её уже никто не читает.
	errChan := make(chan error, 1)

	// ListenAndServe блокирует выполнение, пока сервер работает. Если вызвать его
	// здесь напрямую, код ниже (ожидание сигнала, Shutdown) никогда не выполнится.
	// Поэтому сервер крутится в отдельной горутине, а run идёт дальше и ждёт.
	go func() {
		// Возвращает ошибку всегда: при сбое (например, порт занят) или
		// http.ErrServerClosed после штатного Shutdown.
		err := srv.ListenAndServe()
		errChan <- err
	}()

	// ---- Шаг 7: ожидание и остановка ----
	// select ждёт, какой из каналов отдаст значение первым (как switch для каналов).
	select {
	case err := <-errChan:
		// Сервер упал сам, например не смог занять порт.
		return fmt.Errorf("server: %w", err)
	case <-signalChan:
		// Пришёл Ctrl+C или SIGTERM, начинаем штатную остановку.
		logger.Info("shutting down...")
	}

	// Контекст с таймаутом: даём текущим запросам максимум 5 секунд на завершение.
	// cancel освобождает ресурсы таймера, его принято вызывать всегда через defer.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown перестаёт принимать новые соединения и ждёт, пока текущие запросы
	// доработают (graceful shutdown). Если за 5 секунд не успели, вернёт ошибку.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	logger.Info("server stopped")

	// После return сработают defer в обратном порядке: cancel, db.Close, pool.Close.
	// Сначала остановили сервер, потом закрываем базу: ни один запрос не останется
	// без соединения.
	return nil
}
