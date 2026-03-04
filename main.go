package main

import (
	"fmt"
	"time"

	"dag/internal/dag"
)

func main() {
	fmt.Println("=== DAG 流水线系统示例 ===\n")

	example1()
	fmt.Println("\n" + "==================================================" + "\n")
	example2()
	fmt.Println("\n" + "==================================================" + "\n")
	example3()
}

func example1() {
	fmt.Println("示例 1: 基础流水线执行")
	fmt.Println("---------------------------")

	pipeline := dag.NewPipeline()

	task1 := dag.NewTask("task1", "数据准备", func() error {
		fmt.Println("  [task1] 正在准备数据...")
		time.Sleep(100 * time.Millisecond)
		fmt.Println("  [task1] 数据准备完成")
		return nil
	})

	task2 := dag.NewTask("task2", "数据处理 A", func() error {
		fmt.Println("  [task2] 正在处理数据 A...")
		time.Sleep(200 * time.Millisecond)
		fmt.Println("  [task2] 数据处理 A 完成")
		return nil
	}, "task1")

	task3 := dag.NewTask("task3", "数据处理 B", func() error {
		fmt.Println("  [task3] 正在处理数据 B...")
		time.Sleep(150 * time.Millisecond)
		fmt.Println("  [task3] 数据处理 B 完成")
		return nil
	}, "task1")

	task4 := dag.NewTask("task4", "结果汇总", func() error {
		fmt.Println("  [task4] 正在汇总结果...")
		time.Sleep(100 * time.Millisecond)
		fmt.Println("  [task4] 结果汇总完成")
		return nil
	}, "task2", "task3")

	if err := pipeline.AddTask(task1); err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}
	if err := pipeline.AddTask(task2); err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}
	if err := pipeline.AddTask(task3); err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}
	if err := pipeline.AddTask(task4); err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}

	if err := pipeline.Execute(); err != nil {
		fmt.Printf("\n流水线执行失败: %v\n", err)
	} else {
		fmt.Println("\n流水线执行成功!")
	}

	printSummary(pipeline)
}

func example2() {
	fmt.Println("示例 2: 带重试机制的流水线")
	fmt.Println("---------------------------")

	pipeline := dag.NewPipeline()

	retryCount := 0
	task1 := dag.NewTask("task1", "可能失败的任务", func() error {
		fmt.Printf("  [task1] 尝试执行 (第 %d 次)...\n", retryCount+1)
		retryCount++
		if retryCount < 3 {
			fmt.Println("  [task1] 执行失败，需要重试")
			return fmt.Errorf("临时错误")
		}
		fmt.Println("  [task1] 执行成功!")
		return nil
	})

	task2 := dag.NewTask("task2", "后续任务", func() error {
		fmt.Println("  [task2] 正在执行...")
		time.Sleep(100 * time.Millisecond)
		fmt.Println("  [task2] 执行完成")
		return nil
	}, "task1")

	if err := pipeline.AddTask(task1); err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}
	if err := pipeline.AddTask(task2); err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}

	options := dag.DefaultExecuteOptions()
	options.RetryPolicy = &dag.RetryPolicy{
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		RetryableFunc: func(err error) bool {
			return true
		},
	}

	if err := pipeline.ExecuteWithOptions(options); err != nil {
		fmt.Printf("\n流水线执行失败: %v\n", err)
	} else {
		fmt.Println("\n流水线执行成功!")
	}

	printSummary(pipeline)
}

func example3() {
	fmt.Println("示例 3: 错误处理 (继续执行模式)")
	fmt.Println("---------------------------")

	pipeline := dag.NewPipeline()

	task1 := dag.NewTask("task1", "正常任务", func() error {
		fmt.Println("  [task1] 正在执行...")
		time.Sleep(100 * time.Millisecond)
		fmt.Println("  [task1] 执行完成")
		return nil
	})

	task2 := dag.NewTask("task2", "会失败的任务", func() error {
		fmt.Println("  [task2] 正在执行...")
		time.Sleep(150 * time.Millisecond)
		fmt.Println("  [task2] 执行失败!")
		return fmt.Errorf("任务执行出错")
	}, "task1")

	task3 := dag.NewTask("task3", "独立任务", func() error {
		fmt.Println("  [task3] 正在执行...")
		time.Sleep(200 * time.Millisecond)
		fmt.Println("  [task3] 执行完成")
		return nil
	})

	task4 := dag.NewTask("task4", "依赖失败任务的任务", func() error {
		fmt.Println("  [task4] 应该不会执行...")
		return nil
	}, "task2")

	if err := pipeline.AddTask(task1); err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}
	if err := pipeline.AddTask(task2); err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}
	if err := pipeline.AddTask(task3); err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}
	if err := pipeline.AddTask(task4); err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}

	options := dag.DefaultExecuteOptions()
	options.ContinueOnError = true

	if err := pipeline.ExecuteWithOptions(options); err != nil {
		fmt.Printf("\n流水线执行完成，部分任务失败: %v\n", err)
	} else {
		fmt.Println("\n流水线执行成功!")
	}

	printSummary(pipeline)
}

func printSummary(pipeline *dag.Pipeline) {
	summary := pipeline.GetExecutionSummary()
	fmt.Printf("\n执行摘要:\n")
	fmt.Printf("  总任务数:     %d\n", summary.TotalTasks)
	fmt.Printf("  已完成:       %d\n", summary.CompletedTasks)
	fmt.Printf("  失败:         %d\n", summary.FailedTasks)
	fmt.Printf("  跳过:         %d\n", summary.SkippedTasks)
	fmt.Printf("  总执行时间:   %v\n", summary.ExecutionTime)

	if len(summary.Errors) > 0 {
		fmt.Printf("\n错误详情:\n")
		for _, err := range summary.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}
}
