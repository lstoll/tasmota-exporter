# tasmota-exporter

Simple exporter to scrape data from the Tasmota power sockets I have (https://www.amazon.de/dp/B0CHMMKZCQ)

`go run . -outlets=pinguino:192.168.1.241`

```
# HELP tasmota_apparent_power_voltamperes apparent power of tasmota plug in volt-amperes (VA)
# TYPE tasmota_apparent_power_voltamperes gauge
tasmota_apparent_power_voltamperes{outlet="pinguino"} 1.675
# HELP tasmota_current_amperes current of tasmota plug in ampere (A)
# TYPE tasmota_current_amperes gauge
tasmota_current_amperes{outlet="pinguino"} 0.007
# HELP tasmota_kwh_total total energy usage in kilowatts hours (kWh)
# TYPE tasmota_kwh_total gauge
tasmota_kwh_total{outlet="pinguino"} 0.00367
# HELP tasmota_on Indicates if the tasmota plug is on/off
# TYPE tasmota_on gauge
tasmota_on{outlet="pinguino"} 1
# HELP tasmota_power_factor power factor of tasmota plug
# TYPE tasmota_power_factor gauge
tasmota_power_factor{outlet="pinguino"} 0.12
# HELP tasmota_power_watts current power of tasmota plug in watts (W)
# TYPE tasmota_power_watts gauge
tasmota_power_watts{outlet="pinguino"} 0.2
# HELP tasmota_reactive_power_voltamperesreactive reactive power of tasmota plug in volt-amperes reactive (VAr)
# TYPE tasmota_reactive_power_voltamperesreactive gauge
tasmota_reactive_power_voltamperesreactive{outlet="pinguino"} 1.7
# HELP tasmota_temperature_celsius temperature of the ESP32 chip in celsius
# TYPE tasmota_temperature_celsius gauge
tasmota_temperature_celsius{outlet="pinguino"} 46.3
# HELP tasmota_today_kwh_total todays energy usage total in kilowatts hours (kWh)
# TYPE tasmota_today_kwh_total gauge
tasmota_today_kwh_total{outlet="pinguino"} 0.00367
# HELP tasmota_up Indicates if the tasmota outlet is reachable
# TYPE tasmota_up gauge
tasmota_up{outlet="pinguino"} 1
# HELP tasmota_voltage_volts voltage of tasmota plug in volt (V)
# TYPE tasmota_voltage_volts gauge
tasmota_voltage_volts{outlet="pinguino"} 239
# HELP tasmota_yesterday_kwh_total yesterdays energy usage total in kilowatts hours (kWh)
# TYPE tasmota_yesterday_kwh_total gauge
tasmota_yesterday_kwh_total{outlet="pinguino"} 0
```
