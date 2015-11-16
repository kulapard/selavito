# SelAvito

Утилита для парсинга объявлений (вместе с телефонными номерами) с сайта [avito.ru](https://avito.ru)

**Внимание!** Если проявлять чрезмерную активность, Avito может на время забанить ваш IP.
Чтобы этого не произошло, используйте параметр ```--pause``` (или ```-p```) для указания количества микросекунд между запросами.


## Установка
Скачать и распаковать архив с бинарным файлом:
- OS X - [i386](https://github.com/kulapard/selavito/releases/download/1.0.0/selavito_1.0.0_darwin_i386.tar.gz), [amd64](https://github.com/kulapard/selavito/releases/download/1.0.0/selavito_1.0.0_darwin_amd64.tar.gz)
- Linux - [i386](https://github.com/kulapard/selavito/releases/download/1.0.0/selavito_1.0.0_linux_i386.tar.gz), [amd64](https://github.com/kulapard/selavito/releases/download/1.0.0/selavito_1.0.0_linux_amd64.tar.gz)
- Windows - [i386](https://github.com/kulapard/selavito/releases/download/1.0.0/selavito_1.0.0_windows_i386.tar.gz), [amd64](https://github.com/kulapard/selavito/releases/download/1.0.0/selavito_1.0.0_windows_amd64.tar.gz)

## Запуск
Пример поиска объявления и сбора номеров (но не более 30 номеров) по запросу "кресло" в Москве:
```
selavito -l moskva -q кресло -m 30 --csv=test.csv
```

Ознакомиться со всеми параметрами запуска можно, набрав:
```
selavito -h
```
