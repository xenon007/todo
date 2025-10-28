# Todo Application with projects

## Usage

Run todo or specified params

```
Usage:
  todo --addr :80 --db todo.db --static ./public
```
| Flag       | Description              | Default Value  | Example          |
|------------|--------------------------|----------------|------------------|
| `--addr`   | Server Address with port | `:8080`        | `127.0.0.1:8080` |
| `--db`     | путь к базе данных       | `data/todo.db` | `todo`           |
| `--static` | папка фронтенда          | `web/dist`     | `public`         |

## Development and Build

* The first step for you is to purchase a template.
* Clone repo
* Extract template to `web` folder
* Edit makefile
* Run command `make`