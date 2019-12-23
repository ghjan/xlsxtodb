# Excel(xlsx)导入MySQL/postgres数据库

## 简介

这是工作中用到的一个小工具，将Excel(xlsx)表导入MySQL表中，用Golang写的，每条记录单独一条 `goroutine` 处理，提高效率。
支持随机数生成、密码生成、时间戳；支持关联查询、附表操作等。

## 使用方法

1. 使用命令：
 1.1 `xlsxtodb -driver [mysql/postgres] -dsn [DSN] -table [tablename] -excel [xlsx文件名]
 举例而言：
 ./xlsxtodb -driver mysql -dsn 'root:123456@tcp(127.0.0.1:3306)/xlsxtomysql' -table user -excel demo.xlsx
 这种情况默认只支持一个sheet，读取sheet0
 
 1.2. `xlsxtodb -driver [mysql/postgres] -dsn [DSN] -table [tablename] -excel [xlsx文件名] -sheets [sheetIndex,sheetIndex] -u username -p password -h host -P port -d databasename`
 这种扩展用法，可以支持多个sheet，sheets参数后面的各个sheet用逗号隔开（从0开始），dataStartRow表示数据开始的行数（从1开始）
 -u 数据库用户名
 -p 数据库密码
 -h 数据库host
 -P 数据库端口
 -d 数据库名字
 -sheets 读取的sheets，可以支持多个sheet,各个sheet用逗号隔开（从0开始）
 -dataStartRow 数据开始的行数（从1开始）
 
 举例而言
./xlsxtodb -driver postgres -table categories -excel F:\tmp\models\软件主材明细-categories-20191214.xlsx -u postgres -p postgres -h localhost -P 5432 -d fileserver_nest -sheets 7,8 -dataStartRow 2

./xlsxtodb -driver postgres -table models -excel F:\tmp\models\软件主材明细-models-20191214.xlsx -u postgres -p postgres -h localhost -P 5432 -d fileserver_nest -sheets 0 -dataStartRow 3

## 更新

### v0.2.0
支持多种数据库mysql, postgres
支持多个sheets
go get -u -v github.com/ghjan/xlsxtodb@v0.2.0
### v0.1.2
fork from https://github.com/TargetLiu/xlsxtomysql
修复bug

### v0.1.1

修复bug，优化并发

## DSN

格式：

```
[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
```

示例：

```
root:123@tcp(127.0.0.1:3306)/dbname

```

注意：
Linux、Mac下可能需要输入 `\( \)`

```
go build -o xlsxtodb.exe main.go

./xlsxtodb -driver mysql -dsn 'root:123456@tcp(127.0.0.1:3306)/xlsxtomysql' -table user -excel demo.xlsx

./xlsxtodb -driver postgres -table categories -excel F:\tmp\models\软件主材明细-categories-20191214.xlsx -u postgres -p postgres -h localhost -P 5432 -d fileserver_nest -sheets 7,8 -dataStartRow 2

./xlsxtodb -driver postgres -table models -excel F:\tmp\models\软件主材明细-models-20191214.xlsx -u postgres -p postgres -h localhost -P 5432 -d fileserver_nest -sheets 0 -dataStartRow 3

```

## Excel表导入结构说明

1. 支持多Sheet

2. 第一行对应数据库表字段

 * 通过 `|` 分割

 * `字段名|unique` 判断重复，重复的自动跳过

 * `字段名|password|[md5|bcrypt]` 密码生成，第二个参数[md5|bcrypt]

 * `字段名|find|表名|需要获取的字段|查询字段` 根据内容从其它表查询并获取字段，格式 
 `SELECT 需要获取的字段 FROM 表名 WHERE 查询字段 = 内容`

 * `:other` 附表操作

3. 内容行

 * `:random` 生成随机字符串

 * `:time ` 当前unix时间戳

 * `:null`  空值，自动跳过该值，一般可用于自增id

 * 如果该列为 `password` ，内容为[明文密码]或[明文密码|盐]，盐可以通过[:字段名]获取之前的字段名。
 密码将自动根据字段名中填写的加密方式进行加密

 *  如果该列为 `other` , 格式[表名|字段1|字段2|字段3....] 其它表需要按顺序为每个字段添加内容。
 字段可以为[:null|:id(主表生成的自增ID)|:random|:time]

