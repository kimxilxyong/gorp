@echo off
echo gorp with indexes test batch for windowss
echo please note that only mysql and postgres are tested
echo http://github.com/kimxilxyong/gorp

set GORP_TEST_DSN=gorptest:gorptest@/gorptest?parseTime=true
set GORP_TEST_DIALECT=gomysql
go test .

set GORP_TEST_DSN=user=gorptest password=gorptest dbname=gorptest sslmode=disable
set GORP_TEST_DIALECT=postgres
go test .

