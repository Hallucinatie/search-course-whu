package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

// 全局变量
var (
	coursesData []map[string]any
	coursesMux  sync.RWMutex
	redisClient *redis.Client
	ctx         = context.Background()
)

type SurveyInput struct {
	Curricula   bool    `json:"curricula" binding:"required"`
	Accept      string  `json:"accept" binding:"required,oneof=教学 给分"`
	Expectation float32 `json:"expectation" binding:"required,min=1,max=10"`
	Suggestions string  `json:"suggestions"`
}

type CoursePromotion struct {
	CourseName       string   `json:"course_name" binding:"required"`
	CourseAttribute  string   `json:"course_attribute" binding:"required"`
	ElectiveField    string   `json:"elective_field"`
	Instructor       string   `json:"instructor" binding:"required"`
	Credit           float64  `json:"credit" binding:"required"`
	Content          string   `json:"content" binding:"required"`
	Attendance       string   `json:"attendance" binding:"required"`
	Assessment       string   `json:"assessment" binding:"required"`
	Highlights       string   `json:"highlights" binding:"required"`
	SuitableStudents string   `json:"suitable_students" binding:"required"`
	Resources        []string `json:"resources"`
}

func main() {
	// 初始化Redis客户端
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // 无密码
		DB:       0,  // 默认DB
		PoolSize: 10, // 连接池大小
	})

	// 加载课程数据
	loadCourseData()

	// 清空缓存
	clearCacheOnStartup()

	// 初始化Gin路由
	r := gin.Default()

	// 启用CORS
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type"},
		MaxAge:          86400,
	}))

	// 设置静态文件服务
	r.Use(static.Serve("/", static.LocalFile("./templates", false)))
	r.Use(static.Serve("/static", static.LocalFile("./static", false)))

	// 路由定义
	r.GET("/", serveIndex)
	r.GET("/search", search)
	r.POST("/add_course", addCourse)
	r.GET("/statistic", getStatistics)
	r.GET("/remote_statistic", proxyToRemoteStatistic)
	r.POST("/submit_survey", submitSurvey)
	r.POST("/course_promotion", submitCoursePromotion)

	// 启动服务器
	fmt.Println("Server running on http://0.0.0.0:8082")
	r.Run(":8082")
}

// 在应用启动时清空缓存
func clearCacheOnStartup() {
	log.Println("Clearing caches on startup...")

	// 清空Redis缓存
	err := redisClient.FlushDB(ctx).Err()
	if err != nil {
		log.Printf("Error clearing Redis cache: %v", err)
	}

	log.Println("Caches cleared successfully.")
}

// 加载课程数据
func loadCourseData() {
	coursesMux.Lock()
	defer coursesMux.Unlock()

	// 从CSV文件加载数据
	data, err := CSVFileReader("./CouresesData.csv")
	if err != nil {
		log.Fatalf("Failed to load course data: %v", err)
	}

	coursesData = data
}

// 首页路由处理
func serveIndex(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=360000")
	c.File("./templates/index.html")
}

// 搜索API处理
func search(c *gin.Context) {
	courseName := c.Query("course_name")
	instructor := c.Query("instructor")

	// 创建缓存键
	cacheKey := fmt.Sprintf("search:%s:%s", courseName, instructor)

	// 尝试从Redis获取缓存结果
	cachedResults, err := redisClient.Get(ctx, cacheKey).Bytes()
	var results []map[string]any

	if err == nil {
		// 解析缓存结果
		if err := json.Unmarshal(cachedResults, &results); err != nil {
			log.Printf("Error unmarshaling cached results: %v", err)
			results = searchCourses(courseName, instructor)
		}
	} else {
		// 如果没有缓存或发生错误，执行搜索
		results = searchCourses(courseName, instructor)

		// 缓存结果
		resultBytes, err := json.Marshal(results)
		if err == nil {
			redisClient.Set(ctx, cacheKey, resultBytes, 24*time.Hour)
		}
	}

	// 设置响应头
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=7200") // 缓存2小时

	// 压缩响应
	c.Header("Content-Encoding", "gzip")
	gz := gzip.NewWriter(c.Writer)
	defer gz.Close()

	c.Writer.Header().Del("Content-Length")
	c.Status(http.StatusOK)

	// 写入压缩后的JSON
	json.NewEncoder(gz).Encode(results)
}

// 搜索课程
func searchCourses(courseName, instructor string) []map[string]any {
	coursesMux.RLock()
	defer coursesMux.RUnlock()

	var results []map[string]any

	for _, course := range coursesData {
		nameMatch := true
		instructorMatch := true

		if courseName != "" {
			courseNameValue, _ := course["课程名称"].(string)
			nameMatch = strings.Contains(courseNameValue, courseName)
		}

		if instructor != "" {
			courseInstructor, _ := course["授课老师"].(string)
			instructorMatch = strings.Contains(courseInstructor, instructor)
		}

		if nameMatch && instructorMatch {
			results = append(results, course)
		}
	}

	return results
}

// 添加新课程
func addCourse(c *gin.Context) {
	var newCourse map[string]any
	if err := c.ShouldBindJSON(&newCourse); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证课程数据
	if valid, errMsg := validateCourseData(newCourse); !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	// 保存新课程到CSV
	headers := []string{"course_name", "course_attribute", "instructor", "content", "attendance", "assessment", "grade"}
	fmt.Println(headers)
	err := appendToCSV("./NewCourses.csv", newCourse, headers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Course added successfully"})
}

// 验证课程数据
func validateCourseData(course map[string]any) (bool, string) {
	requiredFields := []string{"course_name", "course_attribute", "instructor", "content", "attendance", "assessment", "grade"}
	for _, field := range requiredFields {
		if _, exists := course[field]; !exists {
			return false, fmt.Sprintf("Missing required field: %s", field)
		}
	}

	if grade, ok := course["grade"].(string); ok && grade != "Unknown" {
		gradeVal, err := strconv.Atoi(grade)
		if err != nil || gradeVal < 0 || gradeVal > 100 {
			return false, "Grade must be between 0 and 100 or 'Unknown'"
		}
	}

	return true, ""
}

// 获取统计信息
func getStatistics(c *gin.Context) {
	coursesMux.RLock()
	coursesCount := len(coursesData)
	coursesMux.RUnlock()

	// 统计新添加的课程数量
	newCoursesCount := 0
	newCoursesPath := "./NewCourses.csv"
	if _, err := os.Stat(newCoursesPath); err == nil {
		newCourses, err := CSVFileReader(newCoursesPath)
		if err == nil {
			newCoursesCount = len(newCourses)
		}
	}

	evaluationCount := coursesCount + newCoursesCount

	// 设置响应
	responseData := gin.H{
		"evaluationCount": evaluationCount,
		"visitCount":      newCoursesCount,
	}

	// 设置头部
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Encoding", "gzip")

	// 创建压缩响应
	c.Writer.Header().Del("Content-Length")
	c.Status(http.StatusOK)

	gz := gzip.NewWriter(c.Writer)
	defer gz.Close()

	json.NewEncoder(gz).Encode(responseData)
}

// 代理转发到远程服务器
func proxyToRemoteStatistic(c *gin.Context) {
	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 创建远程请求
	remoteURL := "http://103.20.220.93:5000/statistic"
	req, err := http.NewRequest("GET", remoteURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建请求失败"})
		return
	}

	// 转发请求头
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "远程请求失败"})
		return
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取响应失败"})
		return
	}

	// 设置响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)
	c.Writer.Write(body)
}

func submitSurvey(c *gin.Context) {
	var input SurveyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "无效的输入数据: " + err.Error()})
		return
	}

	// 准备CSV数据,注意按照正确的列顺序
	surveyData := map[string]any{
		"satisfaction": "", // 保持空值
		"suggestions":  input.Suggestions,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"curricula":    input.Curricula,
		"accept":       input.Accept,
		"expectation":  input.Expectation,
	}

	// CSV文件头部顺序必须匹配
	headers := []string{"satisfaction", "suggestions", "timestamp", "curricula", "accept", "expectation"}

	// 将数据追加到CSV文件
	err := appendToCSV("./surveyData.csv", surveyData, headers)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("保存问卷数据失败: %v", err)})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "问卷提交成功",
		"data":    surveyData,
	})
}

func submitCoursePromotion(c *gin.Context) {
	var input CoursePromotion
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的输入数据: " + err.Error(),
		})
		return
	}

	// 验证选修课领域
	if input.CourseAttribute == "通识选修课（公选课）" && input.ElectiveField == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "通识选修课必须指定选修领域",
		})
		return
	}

	// 准备要保存的数据
	promotionData := map[string]any{
		"course_name":       input.CourseName,
		"course_attribute":  input.CourseAttribute,
		"elective_field":    input.ElectiveField,
		"instructor":        input.Instructor,
		"credit":            input.Credit,
		"content":           input.Content,
		"attendance":        input.Attendance,
		"assessment":        input.Assessment,
		"highlights":        input.Highlights,
		"suitable_students": input.SuitableStudents,
		"resources":         strings.Join(input.Resources, "|"), // 将数组转换为字符串存储
		"submit_time":       time.Now().UTC().Format(time.RFC3339),
	}

	// CSV 文件头
	headers := []string{
		"course_name", "course_attribute", "elective_field", "instructor",
		"credit", "content", "attendance", "assessment", "highlights",
		"suitable_students", "resources", "submit_time",
	}

	// 保存到 CSV 文件
	err := appendToCSV("./coursePromotions.csv", promotionData, headers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("保存课程推广数据失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "课程推广信息已提交",
		"data":    promotionData,
	})
}
