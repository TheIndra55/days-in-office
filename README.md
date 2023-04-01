# days-in-office

`days-in-office` is a small utility to calculate the amount of days you have been in office.

The input used is the takeout from Google Maps Timeline.
You can request it from Google here: https://takeout.google.com/settings/takeout

For this tool you only need the location history JSON file, i.e. the Google Maps Timeline export.
Of course, no data leaves your machine, as the tool only reads the JSON files and processes them locally.

Obviously, "office" could be any location - it simply is my use case in times of working from home.

Once downloaded extract the archive, install the tool and run it.

Installation:

```bash
go install github.com/florianloch/days-in-office@latest
```

Usage:

```shell
days-in-office \
  -start-date "2022-01-02T00:00:00Z" \
  -end-date "2022-12-31T12:59:59Z" \
  -input-dir "./Semantic Location History/" \
  -tolerance 100 \
  -latitude 48.1794935434762 \
  -longitude 11.585803728704384 \
  -print-dates
```

`tolerance` is given in meters and defines the radius around the given coordinates in which the tool will consider a location to be the given target location.
