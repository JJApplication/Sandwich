# Sandwich Structure Package

这个包提供了两个高级数据结构，用于快速高效地处理数据：

## 1. Set 集合

Set 是一个泛型集合，支持任意可比较类型，集合中的元素不会重复。

### 特性
- **泛型支持**: 支持任意可比较类型 (`comparable`)
- **并发安全**: 使用读写锁保证线程安全
- **高效操作**: 基于哈希表实现，查找和插入都是 O(1) 时间复杂度
- **集合运算**: 支持并集、交集、差集操作

### 方法

| 方法 | 描述 | 时间复杂度 |
|------|------|------------|
| `Len()` | 返回集合中元素的数量 | O(1) |
| `Get(item)` | 检查元素是否存在于集合中 | O(1) |
| `Set(item)` | 向集合中添加元素 | O(1) |
| `List()` | 返回集合中所有元素的切片 | O(n) |
| `Find(predicate)` | 查找满足条件的元素 | O(n) |
| `Remove(item)` | 从集合中移除元素 | O(1) |
| `Clear()` | 清空集合 | O(1) |
| `Union(other)` | 返回两个集合的并集 | O(n+m) |
| `Intersection(other)` | 返回两个集合的交集 | O(min(n,m)) |
| `Difference(other)` | 返回两个集合的差集 | O(n) |

### 使用示例

```go
// 创建字符串类型的Set
fruits := NewSet[string]()

// 添加元素
fruits.Add("apple")
fruits.Add("banana")
fruits.Add("orange")

// 检查元素
fmt.Printf("包含apple: %v\n", fruits.Contains("apple")) // true

// 集合运算
citrus := NewSetWithItems("orange", "lemon", "lime")
union := fruits.Union(citrus)
fmt.Printf("并集: %v\n", union.List())
```

## 2. OrderedMap 有序字典

OrderedMap 是一个有序字典，内部元素的顺序是固定的（按照插入顺序）。

### 特性
- **泛型支持**: 键必须是可比较类型，值可以是任意类型
- **插入顺序**: 保持元素的插入顺序
- **并发安全**: 使用读写锁保证线程安全
- **高效操作**: 结合哈希表和双向链表，查找是 O(1)，遍历是 O(n)

### 方法

| 方法 | 描述 | 时间复杂度 |
|------|------|------------|
| `Len()` | 返回字典中元素的数量 | O(1) |
| `Get(key)` | 根据键获取值 | O(1) |
| `Set(key, value)` | 设置键值对 | O(1) |
| `List()` | 返回所有键值对的有序列表 | O(n) |
| `Find(predicate)` | 查找满足条件的键值对 | O(n) |
| `Delete(key)` | 删除指定键的键值对 | O(1) |
| `Keys()` | 返回所有键的有序列表 | O(n) |
| `Values()` | 返回所有值的有序列表 | O(n) |
| `Front()` | 返回第一个键值对 | O(1) |
| `Back()` | 返回最后一个键值对 | O(1) |
| `ForEach(fn)` | 遍历所有键值对 | O(n) |

### 使用示例

```go
// 创建字符串到整数的有序字典
scores := NewOrderedMap[string, int]()

// 添加键值对
scores.Set("Alice", 95)
scores.Set("Bob", 87)
scores.Set("Charlie", 92)

// 获取值
if score, exists := scores.Get("Alice"); exists {
    fmt.Printf("Alice的分数: %d\n", score)
}

// 按插入顺序遍历
scores.ForEach(func(name string, score int) {
    fmt.Printf("%s: %d分\n", name, score)
})
```

## 使用场景

### Set 集合适用场景
- **去重操作**: 快速去除重复元素
- **权限检查**: 检查用户是否拥有特定权限
- **标签管理**: 管理不重复的标签集合
- **集合运算**: 需要进行并集、交集、差集操作

### OrderedMap 有序字典适用场景
- **LRU缓存**: 实现最近最少使用缓存
- **配置管理**: 保持配置项的插入顺序
- **历史记录**: 按时间顺序记录操作历史
- **有序数据**: 需要保持键值对插入顺序的场景

## 性能特点

- **内存效率**: Set使用空结构体节省内存，OrderedMap使用双向链表维护顺序
- **并发安全**: 两个数据结构都使用读写锁，支持多goroutine并发访问
- **类型安全**: 使用Go泛型，编译时类型检查，避免运行时类型错误

## 测试

运行测试：
```bash
go test ./structure -v
```

运行性能测试：
```bash
go test ./structure -bench=.
```

## 注意事项

1. Set的`List()`方法返回的元素顺序不保证，因为内部使用哈希表存储
2. OrderedMap的键必须是可比较类型（支持`==`和`!=`操作）
3. 两个数据结构都是并发安全的，但如果在高并发场景下使用，建议进行性能测试
4. 删除操作会立即释放内存，适合长期运行的应用