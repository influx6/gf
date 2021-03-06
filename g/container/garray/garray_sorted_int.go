// Copyright 2018 gf Author(https://github.com/gogf/gf). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package garray

import (
    "github.com/gogf/gf/g/container/gtype"
    "github.com/gogf/gf/g/internal/rwmutex"
    "github.com/gogf/gf/g/util/gconv"
    "github.com/gogf/gf/g/util/grand"
    "math"
    "sort"
    "strings"
)

// 默认按照从小到大进行排序
type SortedIntArray struct {
    mu          *rwmutex.RWMutex     // 互斥锁
    array       []int                // 底层数组
    unique      *gtype.Bool          // 是否要求不能重复(默认false)
    compareFunc func(v1, v2 int) int // 比较函数，返回值 -1: v1 < v2；0: v1 == v2；1: v1 > v2
}

// Create an empty sorted array.
// The param <unsafe> used to specify whether using array with un-concurrent-safety,
// which is false in default, means concurrent-safe in default.
//
// 创建一个空的排序数组对象，参数unsafe用于指定是否用于非并发安全场景，默认为false，表示并发安全。
func NewSortedIntArray(unsafe...bool) *SortedIntArray {
    return NewSortedIntArraySize(0, unsafe...)
}

// Create a sorted array with given size and cap.
// The param <unsafe> used to specify whether using array with un-concurrent-safety,
// which is false in default, means concurrent-safe in default.
//
// 创建一个指定大小的排序数组对象，参数unsafe用于指定是否用于非并发安全场景，默认为false，表示并发安全。
func NewSortedIntArraySize(cap int, unsafe...bool) *SortedIntArray {
    return &SortedIntArray {
        mu          : rwmutex.New(unsafe...),
        array       : make([]int, 0, cap),
        unique      : gtype.NewBool(),
        compareFunc : func(v1, v2 int) int {
            if v1 < v2 {
                return -1
            }
            if v1 > v2 {
                return 1
            }
            return 0
        },
    }
}

// Create an array with given slice <array>.
// The param <unsafe> used to specify whether using array with un-concurrent-safety,
// which is false in default, means concurrent-safe in default.
//
// 通过给定的slice变量创建排序数组对象，参数unsafe用于指定是否用于非并发安全场景，默认为false，表示并发安全。
func NewSortedIntArrayFrom(array []int, unsafe...bool) *SortedIntArray {
    a := NewSortedIntArraySize(0, unsafe...)
    a.array = array
    sort.Ints(a.array)
    return a
}

// Set the underlying slice array with the given <array> param.
//
// 设置底层数组变量.
func (a *SortedIntArray) SetArray(array []int) *SortedIntArray {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.array = array
    sort.Ints(a.array)
    return a
}

// Sort the array in increasing order.
//
// 将数组排序(默认从低到高).
func (a *SortedIntArray) Sort() *SortedIntArray {
    a.mu.Lock()
    defer a.mu.Unlock()
    sort.Ints(a.array)
    return a
}

// And values to sorted array, the array always keeps sorted.
//
// 添加数据项.
func (a *SortedIntArray) Add(values...int) *SortedIntArray {
    if len(values) == 0 {
        return a
    }
    a.mu.Lock()
    defer a.mu.Unlock()
    for _, value := range values {
        index, cmp := a.binSearch(value, false)
        if a.unique.Val() && cmp == 0 {
            continue
        }
        if index < 0 {
            a.array = append(a.array, value)
            continue
        }
        // 加到指定索引后面
        if cmp > 0 {
            index++
        }
        rear   := append([]int{}, a.array[index : ]...)
        a.array = append(a.array[0 : index], value)
        a.array = append(a.array, rear...)
    }
    return a
}

// Get value by index.
//
// 获取指定索引的数据项, 调用方注意判断数组边界。
func (a *SortedIntArray) Get(index int) int {
    a.mu.RLock()
    defer a.mu.RUnlock()
    value := a.array[index]
    return value
}

// Remove an item by index.
//
// 删除指定索引的数据项, 调用方注意判断数组边界。
func (a *SortedIntArray) Remove(index int) int {
    a.mu.Lock()
    defer a.mu.Unlock()
    // 边界删除判断，以提高删除效率
    if index == 0 {
        value  := a.array[0]
        a.array = a.array[1 : ]
        return value
    } else if index == len(a.array) - 1 {
        value  := a.array[index]
        a.array = a.array[: index]
        return value
    }
    // 如果非边界删除，会涉及到数组创建，那么删除的效率差一些
    value  := a.array[index]
    a.array = append(a.array[ : index], a.array[index + 1 : ]...)
    return value
}

// Push new items to the beginning of array.
//
// 将数据项添加到数组的最左端(索引为0)。
func (a *SortedIntArray) PopLeft() int {
    a.mu.Lock()
    defer a.mu.Unlock()
    value  := a.array[0]
    a.array = a.array[1 : ]
    return value
}

// Push new items to the end of array.
//
// 将数据项添加到数组的最右端(索引为length - 1)。
func (a *SortedIntArray) PopRight() int {
    a.mu.Lock()
    defer a.mu.Unlock()
    index  := len(a.array) - 1
    value  := a.array[index]
    a.array = a.array[: index]
    return value
}

// Pop an random item from array.
//
// 随机将一个数据项移出数组，并返回该数据项。
func (a *SortedIntArray) PopRand() int {
    return a.Remove(grand.Intn(len(a.array)))
}

// Pop <size> items from the beginning of array.
//
// 将最左端(首部)的size个数据项移出数组，并返回该数据项
func (a *SortedIntArray) PopLefts(size int) []int {
    a.mu.Lock()
    defer a.mu.Unlock()
    length := len(a.array)
    if size > length {
        size = length
    }
    value  := a.array[0 : size]
    a.array = a.array[size : ]
    return value
}

// Pop <size> items from the end of array.
//
// 将最右端(尾部)的size个数据项移出数组，并返回该数据项
func (a *SortedIntArray) PopRights(size int) []int {
    a.mu.Lock()
    defer a.mu.Unlock()
    index := len(a.array) - size
    if index < 0 {
        index = 0
    }
    value  := a.array[index :]
    a.array = a.array[ : index]
    return value
}

// Get items by range, returns array[start:end].
// Be aware that, if in concurrent-safe usage, it returns a copy of slice;
// else a pointer to the underlying data.
//
// 将最右端(尾部)的size个数据项移出数组，并返回该数据项
func (a *SortedIntArray) Range(start, end int) []int {
    a.mu.RLock()
    defer a.mu.RUnlock()
    length := len(a.array)
    if start > length || start > end {
        return nil
    }
    if start < 0 {
        start = 0
    }
    if end > length {
        end = length
    }
    array  := ([]int)(nil)
    if a.mu.IsSafe() {
        a.mu.RLock()
        defer a.mu.RUnlock()
        array = make([]int, end - start)
        copy(array, a.array[start : end])
    } else {
        array = a.array[start : end]
    }
    return array
}

// Get the length of array.
//
// 数组长度。
func (a *SortedIntArray) Len() int {
    a.mu.RLock()
    length := len(a.array)
    a.mu.RUnlock()
    return length
}

// Calculate the sum of values in an array.
//
// 对数组中的元素项求和。
func (a *SortedIntArray) Sum() (sum int) {
    a.mu.RLock()
    defer a.mu.RUnlock()
    for _, v := range a.array {
        sum += v
    }
    return
}

// Get the underlying data of array.
// Be aware that, if in concurrent-safe usage, it returns a copy of slice;
// else a pointer to the underlying data.
//
// 返回原始数据数组.
func (a *SortedIntArray) Slice() []int {
    array := ([]int)(nil)
    if a.mu.IsSafe() {
        a.mu.RLock()
        defer a.mu.RUnlock()
        array = make([]int, len(a.array))
        copy(array, a.array)
    } else {
        array = a.array
    }
    return array
}

// Check whether a value exists in the array.
//
// 查找指定数值是否存在。
func (a *SortedIntArray) Contains(value int) bool {
    return a.Search(value) == 0
}

// Search array by <value>, returns the index of <value>, returns -1 if not exists.
//
// 查找指定数值的索引位置，返回索引位置，如果查找不到则返回-1。
func (a *SortedIntArray) Search(value int) (index int) {
    index, _ = a.binSearch(value, true)
    return
}

// Binary search.
//
// 二分查找.
func (a *SortedIntArray) binSearch(value int, lock bool) (index int, result int) {
    if len(a.array) == 0 {
        return -1, -2
    }
    if lock {
        a.mu.RLock()
        defer a.mu.RUnlock()
    }
    min := 0
    max := len(a.array) - 1
    mid := 0
    cmp := -2
    for min <= max {
        mid = int((min + max) / 2)
        cmp = a.compareFunc(value, a.array[mid])
        switch {
            case cmp < 0 : max = mid - 1
            case cmp > 0 : min = mid + 1
            default :
                return mid, cmp
        }
    }
    return mid, cmp
}

// Set unique mark to the array,
// which means it does not contain any repeated items.
// It also do unique check, remove all repeated items.
//
// 设置是否允许数组唯一.
func (a *SortedIntArray) SetUnique(unique bool) *SortedIntArray {
    oldUnique := a.unique.Val()
    a.unique.Set(unique)
    if unique && oldUnique != unique {
        a.Unique()
    }
    return a
}

// Do unique check, remove all repeated items.
//
// 清理数组中重复的元素项.
func (a *SortedIntArray) Unique() *SortedIntArray {
    a.mu.Lock()
    i := 0
    for {
        if i == len(a.array) - 1 {
            break
        }
        if a.compareFunc(a.array[i], a.array[i + 1]) == 0 {
            a.array = append(a.array[ : i + 1], a.array[i + 1 + 1 : ]...)
        } else {
            i++
        }
    }
    a.mu.Unlock()
    return a
}

// Return a new array, which is a copy of current array.
//
// 克隆当前数组，返回当前数组的一个拷贝。
func (a *SortedIntArray) Clone() (newArray *SortedIntArray) {
    a.mu.RLock()
    array := make([]int, len(a.array))
    copy(array, a.array)
    a.mu.RUnlock()
    return NewSortedIntArrayFrom(array, !a.mu.IsSafe())
}

// Clear array.
//
// 清空数据数组。
func (a *SortedIntArray) Clear() *SortedIntArray {
    a.mu.Lock()
    if len(a.array) > 0 {
        a.array = make([]int, 0)
    }
    a.mu.Unlock()
    return a
}

// Lock writing by callback function f.
//
// 使用自定义方法执行加锁修改操作。
func (a *SortedIntArray) LockFunc(f func(array []int)) *SortedIntArray {
    a.mu.Lock(true)
    defer a.mu.Unlock(true)
    f(a.array)
    return a
}

// Lock reading by callback function f.
//
// 使用自定义方法执行加锁读取操作。
func (a *SortedIntArray) RLockFunc(f func(array []int)) *SortedIntArray {
    a.mu.RLock(true)
    defer a.mu.RUnlock(true)
    f(a.array)
    return a
}

// Merge two arrays.
//
// 合并两个数组.
func (a *SortedIntArray) Merge(array *SortedIntArray) *SortedIntArray {
    a.mu.Lock()
    defer a.mu.Unlock()
    if a != array {
        array.mu.RLock()
        defer array.mu.RUnlock()
    }
    a.array = append(a.array, array.array...)
    sort.Ints(a.array)
    return a
}

// Chunks an array into arrays with size elements.
// The last chunk may contain less than size elements.
//
// 将一个数组分割成多个数组，其中每个数组的单元数目由size决定。最后一个数组的单元数目可能会少于size个。
func (a *SortedIntArray) Chunk(size int) [][]int {
    if size < 1 {
        return nil
    }
    a.mu.RLock()
    defer a.mu.RUnlock()
    length := len(a.array)
    chunks := int(math.Ceil(float64(length) / float64(size)))
    var n [][]int
    for i, end := 0, 0; chunks > 0; chunks-- {
        end = (i + 1) * size
        if end > length {
            end = length
        }
        n = append(n, a.array[i*size : end])
        i++
    }
    return n
}

// Extract a slice of the array(If in concurrent safe usage,
// it returns a copy of the slice; else a pointer).
// It returns the sequence of elements from the array array as specified
// by the offset and length parameters.
//
// 返回根据offset和size参数所指定的数组中的一段序列。
func (a *SortedIntArray) SubSlice(offset, size int) []int {
    a.mu.RLock()
    defer a.mu.RUnlock()
    if offset > len(a.array) {
        return nil
    }
    if offset + size > len(a.array) {
        size = len(a.array) - offset
    }
    if a.mu.IsSafe() {
        s := make([]int, size)
        copy(s, a.array[offset:])
        return s
    } else {
        return a.array[offset:]
    }
}

// Picks one or more random entries out of an array(a copy),
// and returns the key (or keys) of the random entries.
//
// 从数组中随机取出size个元素项，构成slice返回。
func (a *SortedIntArray) Rand(size int) []int {
    a.mu.RLock()
    defer a.mu.RUnlock()
    if size > len(a.array) {
        size = len(a.array)
    }
    n := make([]int, size)
    for i, v := range grand.Perm(len(a.array)) {
        n[i] = a.array[v]
        if i == size - 1 {
            break
        }
    }
    return n
}

// Join array elements with a string.
//
// 使用glue字符串串连当前数组的元素项，构造成新的字符串返回。
func (a *SortedIntArray) Join(glue string) string {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return strings.Join(gconv.Strings(a.array), glue)
}