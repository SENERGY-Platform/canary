# Canary

- imitates user behavior to test if components of the platform are running correctly
- tests are started by http requests to GET /metrics
- GET /metrics returns prometheus metrics
- the tests will create a canary device-type and device, if they don't already exist