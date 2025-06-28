/**
 * 统一加载状态管理Hook
 * 提供一致的加载状态处理和显示
 */

import { useState, useCallback, useRef, useEffect } from 'react';
import { toast } from 'sonner';

// 加载状态类型
export interface LoadingState {
  isLoading: boolean;
  message: string;
  progress?: number;
  startTime: number | null;
  duration: number;
}

// 加载配置
interface LoadingConfig {
  message?: string;
  showToast?: boolean;
  minDuration?: number; // 最小显示时间（毫秒）
  maxDuration?: number; // 最大显示时间（毫秒）
  showProgress?: boolean;
  autoHide?: boolean;
}

const DEFAULT_CONFIG: LoadingConfig = {
  message: '加载中...',
  showToast: false,
  minDuration: 300,
  maxDuration: 30000,
  showProgress: false,
  autoHide: true,
};

// 基础加载Hook
export function useLoading(config: LoadingConfig = {}) {
  const finalConfig = { ...DEFAULT_CONFIG, ...config };
  const [state, setState] = useState<LoadingState>({
    isLoading: false,
    message: finalConfig.message || '加载中...',
    progress: undefined,
    startTime: null,
    duration: 0,
  });

  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  // 开始加载
  const startLoading = useCallback(
    (message?: string) => {
      const startTime = Date.now();

      setState({
        isLoading: true,
        message: message || finalConfig.message || '加载中...',
        progress: finalConfig.showProgress ? 0 : undefined,
        startTime,
        duration: 0,
      });

      // 显示Toast
      if (finalConfig.showToast) {
        toast.loading(message || finalConfig.message || '加载中...');
      }

      // 设置最大持续时间
      if (finalConfig.maxDuration) {
        timeoutRef.current = setTimeout(() => {
          stopLoading();
          console.warn('Loading timeout reached');
        }, finalConfig.maxDuration);
      }

      // 更新持续时间
      intervalRef.current = setInterval(() => {
        setState((prev) => ({
          ...prev,
          duration: prev.startTime ? Date.now() - prev.startTime : 0,
        }));
      }, 100);
    },
    [finalConfig]
  );

  // 停止加载
  const stopLoading = useCallback(async () => {
    const currentState = state;

    // 清除定时器
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
    }

    // 确保最小显示时间
    if (currentState.startTime && finalConfig.minDuration) {
      const elapsed = Date.now() - currentState.startTime;
      if (elapsed < finalConfig.minDuration) {
        await new Promise((resolve) => setTimeout(resolve, finalConfig.minDuration! - elapsed));
      }
    }

    setState((prev) => ({
      ...prev,
      isLoading: false,
      startTime: null,
    }));

    // 隐藏Toast
    if (finalConfig.showToast) {
      toast.dismiss();
    }
  }, [state, finalConfig]);

  // 更新进度
  const updateProgress = useCallback((progress: number) => {
    setState((prev) => ({
      ...prev,
      progress: Math.max(0, Math.min(100, progress)),
    }));
  }, []);

  // 更新消息
  const updateMessage = useCallback((message: string) => {
    setState((prev) => ({
      ...prev,
      message,
    }));
  }, []);

  // 清理
  useEffect(() => {
    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, []);

  return {
    ...state,
    startLoading,
    stopLoading,
    updateProgress,
    updateMessage,
  };
}

// 异步操作加载Hook
export function useAsyncLoading<T = any>(config: LoadingConfig = {}) {
  const loading = useLoading(config);

  const execute = useCallback(
    async (asyncFn: () => Promise<T>, loadingMessage?: string): Promise<T | null> => {
      try {
        loading.startLoading(loadingMessage);
        const result = await asyncFn();
        return result;
      } catch (error) {
        console.error('Async operation failed:', error);
        throw error;
      } finally {
        await loading.stopLoading();
      }
    },
    [loading]
  );

  return {
    ...loading,
    execute,
  };
}

// 多步骤加载Hook
export function useStepLoading(steps: string[], config: LoadingConfig = {}) {
  const [currentStep, setCurrentStep] = useState(0);
  const [completedSteps, setCompletedSteps] = useState<boolean[]>(
    new Array(steps.length).fill(false)
  );

  const loading = useLoading({
    ...config,
    showProgress: true,
  });

  const startStep = useCallback(
    (stepIndex: number) => {
      if (stepIndex >= 0 && stepIndex < steps.length) {
        setCurrentStep(stepIndex);
        loading.startLoading(steps[stepIndex]);
        loading.updateProgress((stepIndex / steps.length) * 100);
      }
    },
    [steps, loading]
  );

  const completeStep = useCallback(
    (stepIndex: number) => {
      setCompletedSteps((prev) => {
        const newSteps = [...prev];
        newSteps[stepIndex] = true;
        return newSteps;
      });

      const nextStep = stepIndex + 1;
      if (nextStep < steps.length) {
        startStep(nextStep);
      } else {
        loading.updateProgress(100);
        setTimeout(() => loading.stopLoading(), 500);
      }
    },
    [steps.length, startStep, loading]
  );

  const reset = useCallback(() => {
    setCurrentStep(0);
    setCompletedSteps(new Array(steps.length).fill(false));
    loading.stopLoading();
  }, [steps.length, loading]);

  return {
    ...loading,
    currentStep,
    completedSteps,
    totalSteps: steps.length,
    currentStepName: steps[currentStep],
    startStep,
    completeStep,
    reset,
    isComplete: completedSteps.every(Boolean),
  };
}

// 全局加载状态管理
class GlobalLoadingManager {
  private loadingStates = new Map<string, LoadingState>();
  private listeners = new Set<(states: Map<string, LoadingState>) => void>();

  // 添加加载状态
  addLoading(key: string, state: LoadingState) {
    this.loadingStates.set(key, state);
    this.notifyListeners();
  }

  // 移除加载状态
  removeLoading(key: string) {
    this.loadingStates.delete(key);
    this.notifyListeners();
  }

  // 获取所有加载状态
  getAllStates() {
    return new Map(this.loadingStates);
  }

  // 检查是否有任何加载中的状态
  hasAnyLoading() {
    return Array.from(this.loadingStates.values()).some((state) => state.isLoading);
  }

  // 添加监听器
  addListener(listener: (states: Map<string, LoadingState>) => void): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  // 通知监听器
  private notifyListeners() {
    this.listeners.forEach((listener) => listener(this.getAllStates()));
  }
}

const globalLoadingManager = new GlobalLoadingManager();

// 全局加载Hook
export function useGlobalLoading(key: string, config: LoadingConfig = {}) {
  const loading = useLoading(config);

  useEffect(() => {
    if (loading.isLoading) {
      globalLoadingManager.addLoading(key, loading);
    } else {
      globalLoadingManager.removeLoading(key);
    }
  }, [key, loading]);

  useEffect(() => {
    return () => {
      globalLoadingManager.removeLoading(key);
    };
  }, [key]);

  return loading;
}

// 监听全局加载状态Hook
export function useGlobalLoadingState() {
  const [states, setStates] = useState<Map<string, LoadingState>>(
    globalLoadingManager.getAllStates()
  );

  useEffect(() => {
    return globalLoadingManager.addListener(setStates);
  }, []);

  return {
    states,
    hasAnyLoading: globalLoadingManager.hasAnyLoading(),
    loadingCount: states.size,
  };
}

// 页面加载Hook
export function usePageLoading() {
  const [isPageLoading, setIsPageLoading] = useState(true);
  const [loadingMessage, setLoadingMessage] = useState('页面加载中...');

  const setPageLoading = useCallback((loading: boolean, message?: string) => {
    setIsPageLoading(loading);
    if (message) {
      setLoadingMessage(message);
    }
  }, []);

  // 页面加载完成后自动隐藏
  useEffect(() => {
    const timer = setTimeout(() => {
      setIsPageLoading(false);
    }, 100);

    return () => clearTimeout(timer);
  }, []);

  return {
    isPageLoading,
    loadingMessage,
    setPageLoading,
  };
}

// 懒加载Hook
export function useLazyLoading<T>(loadFn: () => Promise<T>, deps: any[] = []) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const loading = useLoading({ message: '正在加载...' });

  const load = useCallback(async () => {
    try {
      loading.startLoading();
      setError(null);
      const result = await loadFn();
      setData(result);
      return result;
    } catch (err) {
      const error = err instanceof Error ? err : new Error('加载失败');
      setError(error);
      throw error;
    } finally {
      loading.stopLoading();
    }
  }, deps);

  return {
    data,
    error,
    load,
    ...loading,
  };
}
