package main
import (
  "context"
  "database/sql"
  "fmt"
  _ "github.com/lib/pq"
)
func main(){
  dsn := "postgres://aether:aether@localhost:5432/aethernet?sslmode=disable"
  db, err := sql.Open("postgres", dsn)
  if err != nil { panic(err) }
  defer db.Close()
  if err := db.PingContext(context.Background()); err != nil { panic(err) }
  var user, dbname string
  if err := db.QueryRowContext(context.Background(), "select current_user, current_database()").Scan(&user, &dbname); err != nil { panic(err) }
  fmt.Println(user, dbname)
}
