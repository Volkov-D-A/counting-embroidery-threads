# Подсчет нитей DMC

Кроссплатформенное десктоп-приложение для расчета расхода нитей DMC по TXT-отчетам,
выгруженным из сторонней программы для вышивки.

Планируемый стек:

- Wails v2.12.0
- Go
- React + TypeScript
- Vite

Согласованная спецификация находится в [docs/spec.md](docs/spec.md).

## Разработка

```bash
GOCACHE=/tmp/go-build-cache go test ./...
cd frontend
npm install
npm run build
```

Полная Linux-сборка на системах с `webkit2gtk-4.1`:

```bash
GOCACHE=/tmp/go-build-cache wails build -tags webkit2_41
```
