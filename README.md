# 武汉大学课程评价搜索系统

这是一个使用 Go 语言实现的武汉大学课程评价搜索系统后端，保持与原 Python 实现的 API 兼容。

## 功能特性

- 课程搜索：支持按课程名称和教师名称搜索
- 课程评价提交：允许用户提交新的课程评价
- 统计信息：提供评价总数和访问量等统计数据
- Redis 缓存：优化热门搜索查询性能

## 技术栈

- Go 1.24+
- Gin Web 框架
- Redis 缓存

## 性能优势

相比原 Python 版本，Go 语言实现有以下优势：

1. **更高性能**：Go 的并发处理能力更强，可以处理更多并发请求
2. **更低内存占用**：Go 程序内存占用更少，适合长时间运行
3. **更快的启动时间**：Go 编译为单一二进制文件，启动速度快
4. **依然保持 Redis 缓存**：保留了 Redis 缓存功能，确保热点查询高效

## 项目结构

```
search-course-whu/
├── main.go          # 主程序入口
├── csv_utils.go     # CSV文件处理工具
├── go.mod           # Go模块文件
└── README.md        # 项目说明文档
```

## 使用方法

### 前提条件

- Go 1.24 或更高版本
- Redis 服务器

### 安装依赖

```bash
go get github.com/gin-gonic/gin github.com/gin-contrib/cors github.com/gin-contrib/static github.com/go-redis/redis/v8 golang.org/x/text/encoding/simplifiedchinese
```

### 运行应用

```bash
go run *.go
```

服务器将在 http://localhost:8082 上运行。


## API 接口

### 1. 课程搜索

```
GET /search?course_name=<课程名>&instructor=<教师名>
```

### 2. 提交课程评价

```
POST /add_course
```

请求体示例:

```json
{
  "course_name": "课程名称",
  "course_attribute": "课程属性",
  "instructor": "教师名称",
  "content": "课程内容",
  "attendance": "考勤情况",
  "assessment": "考核方式",
  "grade": "成绩"
}
```

### 3. 获取统计信息

```
GET /statistic
```

## 部署

### 构建二进制文件

```bash
go build -o search-course-whu
```

这将创建一个名为 search-course-whu 的二进制可执行文件。
