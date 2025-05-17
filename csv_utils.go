package main

import (
    "bytes"
    "encoding/csv"
    "fmt"
    "os"
)

// CSVFileReader 读取CSV文件并将其转换为字典切片
func CSVFileReader(filePath string) ([]map[string]any, error) {
    // 读取UTF-8编码的文件
    content, err := os.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("无法打开CSV文件: %v", err)
    }

    // 创建CSV读取器
    reader := csv.NewReader(bytes.NewReader(content))
    // 设置一些宽松的解析选项，以处理格式不严格的CSV
    reader.LazyQuotes = true
    reader.FieldsPerRecord = -1 // 允许每行有不同数量的字段

    // 读取所有记录
    records, err := reader.ReadAll()
    if err != nil {
        return nil, fmt.Errorf("读取CSV文件失败: %v", err)
    }

    if len(records) == 0 {
        return []map[string]any{}, nil
    }

    // 获取头部（列名）
    headers := records[0]

    // 遍历数据行并创建字典
    var result []map[string]any
    for i := 1; i < len(records); i++ {
        row := records[i]
        item := make(map[string]any)

        for j, value := range row {
            if j < len(headers) {
                // 处理空值
                if value == "" {
                    value = "未知"
                }
                item[headers[j]] = value
            }
        }

        result = append(result, item)
    }

    return result, nil
}

// 将数据写入CSV文件
func writeCSV(filePath string, data []map[string]any, headers []string) error {
    // 创建文件
    file, err := os.Create(filePath)
    if err != nil {
        return fmt.Errorf("创建CSV文件失败: %v", err)
    }
    defer file.Close()

    // 创建CSV写入器
    writer := csv.NewWriter(file)

    // 写入头部
    if err := writer.Write(headers); err != nil {
        return fmt.Errorf("写入CSV头部失败: %v", err)
    }

    // 写入数据行
    for _, item := range data {
        var row []string
        for _, header := range headers {
            if val, ok := item[header]; ok {
                row = append(row, fmt.Sprintf("%v", val))
            } else {
                row = append(row, "未知")
            }
        }
        if err := writer.Write(row); err != nil {
            return fmt.Errorf("写入CSV数据失败: %v", err)
        }
    }

    writer.Flush()
    return writer.Error()
}

// appendToCSV 添加单行数据到现有CSV或创建新CSV
func appendToCSV(filePath string, data map[string]any, headers []string) error {
    // 检查文件是否存在
    fileExists := true
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        fileExists = false
    }

    // 如果文件不存在，创建新文件并写入标题和数据
    if !fileExists {
        return writeCSV(filePath, []map[string]any{data}, headers)
    }

    // 如果文件存在，直接追加数据
    file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("打开CSV文件失败: %v", err)
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    
    // 构建行数据
    var row []string
    for _, header := range headers {
        if val, ok := data[header]; ok {
            row = append(row, fmt.Sprintf("%v", val))
        } else {
            row = append(row, "未知")
        }
    }
    
    // 写入单行数据
    if err := writer.Write(row); err != nil {
        return fmt.Errorf("追加CSV数据失败: %v", err)
    }
    
    writer.Flush()
    return writer.Error()
}