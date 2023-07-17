# Example Data

Emergency response data from NOAA is included as a small test example used for development and testing.

To import the data:

```bash
export DSN=postgresql://<username>:<password>@localhost:5439/stac
pypgstac load collections noaa-emergency-response.json --dsn $DSN --method insert 
pypgstac load items noaa-eri-nashville2020.json --dsn $DSN --method insert
```