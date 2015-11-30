# SelAvito

[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/kulapard/selavito/blob/master/LICENSE)
[![Build Status](https://travis-ci.org/kulapard/selavito.svg?branch=master)](https://travis-ci.org/kulapard/selavito)
[![Code Health](https://landscape.io/github/kulapard/selavito/master/landscape.svg?style=flat)](https://landscape.io/github/kulapard/selavito/master)

Утилита для парсинга объявлений (вместе с телефонными номерами) с сайта [avito.ru](https://avito.ru)

**Внимание!** Если проявлять чрезмерную активность, Avito может на время забанить ваш IP.
Чтобы этого не произошло, используйте параметр ```--pause``` (или ```-p```) для указания количества микросекунд между запросами.


## Установка
[Скачать](https://github.com/kulapard/selavito/releases/latest) и распаковать архив с бинарным файлом.

## Запуск
Пример поиска объявления и сбора номеров (но не более 30 номеров) по запросу "кресло" в Москве:
```
selavito -l moskva -q кресло -m 30 --csv=test.csv
```

Ознакомиться со всеми параметрами запуска можно, набрав:
```
selavito -h
```

## Лицензионное соглашение
Если коротко, то что хотите, то и делайте, но автор ответвтвенности за последствия использования программы не несёт. Подробнее читайте [тут](https://github.com/kulapard/selavito/blob/master/LICENSE).
