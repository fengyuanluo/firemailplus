/**
 * 统一错误处理机制
 * 提供一致的错误分类、处理和显示
 */

import { toast } from 'sonner';
import React from 'react';

// 错误类型枚举
export enum ErrorType {
  NETWORK = 'NETWORK',
  AUTH = 'AUTH',
  VALIDATION = 'VALIDATION',
  PERMISSION = 'PERMISSION',
  NOT_FOUND = 'NOT_FOUND',
  SERVER = 'SERVER',
  UNKNOWN = 'UNKNOWN',
}

// 错误严重程度
export enum ErrorSeverity {
  LOW = 'LOW',
  MEDIUM = 'MEDIUM',
  HIGH = 'HIGH',
  CRITICAL = 'CRITICAL',
}

// 标准化错误接口
export interface AppError {
  type: ErrorType;
  severity: ErrorSeverity;
  message: string;
  code?: string | number;
  details?: any;
  timestamp: number;
  context?: string;
}

// 错误处理配置
interface ErrorHandlerConfig {
  showToast: boolean;
  logToConsole: boolean;
  reportToService: boolean;
  autoRetry: boolean;
  maxRetries: number;
}

const DEFAULT_CONFIG: ErrorHandlerConfig = {
  showToast: true,
  logToConsole: true,
  reportToService: false,
  autoRetry: false,
  maxRetries: 0,
};

// 错误分类器
export class ErrorClassifier {
  static classify(error: any): AppError {
    const timestamp = Date.now();

    // 网络错误
    if (error.name === 'NetworkError' || error.code === 'NETWORK_ERROR') {
      return {
        type: ErrorType.NETWORK,
        severity: ErrorSeverity.MEDIUM,
        message: '网络连接失败，请检查网络设置',
        code: error.code,
        details: error,
        timestamp,
        context: 'network',
      };
    }

    // HTTP状态码错误
    if (error.status || error.response?.status) {
      const status = error.status || error.response?.status;

      switch (status) {
        case 401:
          return {
            type: ErrorType.AUTH,
            severity: ErrorSeverity.HIGH,
            message: '登录已过期，请重新登录',
            code: status,
            details: error,
            timestamp,
            context: 'auth',
          };

        case 403:
          return {
            type: ErrorType.PERMISSION,
            severity: ErrorSeverity.MEDIUM,
            message: '权限不足，无法执行此操作',
            code: status,
            details: error,
            timestamp,
            context: 'permission',
          };

        case 404:
          return {
            type: ErrorType.NOT_FOUND,
            severity: ErrorSeverity.LOW,
            message: '请求的资源不存在',
            code: status,
            details: error,
            timestamp,
            context: 'not_found',
          };

        case 422:
          return {
            type: ErrorType.VALIDATION,
            severity: ErrorSeverity.LOW,
            message: error.message || '输入数据格式错误',
            code: status,
            details: error,
            timestamp,
            context: 'validation',
          };

        case 500:
        case 502:
        case 503:
          return {
            type: ErrorType.SERVER,
            severity: ErrorSeverity.HIGH,
            message: '服务器暂时不可用，请稍后重试',
            code: status,
            details: error,
            timestamp,
            context: 'server',
          };

        default:
          return {
            type: ErrorType.UNKNOWN,
            severity: ErrorSeverity.MEDIUM,
            message: error.message || '发生未知错误',
            code: status,
            details: error,
            timestamp,
            context: 'http',
          };
      }
    }

    // 验证错误
    if (error.name === 'ValidationError' || error.type === 'validation') {
      return {
        type: ErrorType.VALIDATION,
        severity: ErrorSeverity.LOW,
        message: error.message || '数据验证失败',
        code: error.code,
        details: error,
        timestamp,
        context: 'validation',
      };
    }

    // 默认错误
    return {
      type: ErrorType.UNKNOWN,
      severity: ErrorSeverity.MEDIUM,
      message: error.message || '发生未知错误',
      code: error.code,
      details: error,
      timestamp,
      context: 'unknown',
    };
  }
}

// 错误处理器
export class ErrorHandler {
  private config: ErrorHandlerConfig;
  private errorHistory: AppError[] = [];
  private maxHistorySize = 50;

  constructor(config: Partial<ErrorHandlerConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  // 处理错误
  handle(error: any, context?: string, config?: Partial<ErrorHandlerConfig>): AppError {
    const finalConfig = { ...this.config, ...config };
    const appError = ErrorClassifier.classify(error);

    if (context) {
      appError.context = context;
    }

    // 记录错误历史
    this.addToHistory(appError);

    // 控制台日志
    if (finalConfig.logToConsole) {
      this.logError(appError);
    }

    // 显示Toast
    if (finalConfig.showToast) {
      this.showToast(appError);
    }

    // 上报错误（如果配置了）
    if (finalConfig.reportToService) {
      this.reportError(appError);
    }

    return appError;
  }

  // 记录错误历史
  private addToHistory(error: AppError) {
    this.errorHistory.unshift(error);
    if (this.errorHistory.length > this.maxHistorySize) {
      this.errorHistory = this.errorHistory.slice(0, this.maxHistorySize);
    }
  }

  // 控制台日志
  private logError(error: AppError) {
    const logLevel = this.getLogLevel(error.severity);
    console[logLevel](`[${error.type}] ${error.message}`, {
      code: error.code,
      context: error.context,
      details: error.details,
      timestamp: new Date(error.timestamp).toISOString(),
    });
  }

  // 获取日志级别
  private getLogLevel(severity: ErrorSeverity): 'log' | 'warn' | 'error' {
    switch (severity) {
      case ErrorSeverity.LOW:
        return 'log';
      case ErrorSeverity.MEDIUM:
        return 'warn';
      case ErrorSeverity.HIGH:
      case ErrorSeverity.CRITICAL:
        return 'error';
      default:
        return 'warn';
    }
  }

  // 显示Toast通知
  private showToast(error: AppError) {
    const toastOptions = {
      duration: this.getToastDuration(error.severity),
    };

    switch (error.severity) {
      case ErrorSeverity.LOW:
        toast.info(error.message, toastOptions);
        break;
      case ErrorSeverity.MEDIUM:
        toast.warning(error.message, toastOptions);
        break;
      case ErrorSeverity.HIGH:
      case ErrorSeverity.CRITICAL:
        toast.error(error.message, toastOptions);
        break;
      default:
        toast.error(error.message, toastOptions);
    }
  }

  // 获取Toast持续时间
  private getToastDuration(severity: ErrorSeverity): number {
    switch (severity) {
      case ErrorSeverity.LOW:
        return 3000;
      case ErrorSeverity.MEDIUM:
        return 4000;
      case ErrorSeverity.HIGH:
        return 6000;
      case ErrorSeverity.CRITICAL:
        return 8000;
      default:
        return 4000;
    }
  }

  // 上报错误到服务
  private async reportError(error: AppError) {
    try {
      // 这里可以实现错误上报逻辑
      console.log('Error reported:', error);
    } catch (reportError) {
      console.error('Failed to report error:', reportError);
    }
  }

  // 获取错误历史
  getErrorHistory(): AppError[] {
    return [...this.errorHistory];
  }

  // 清除错误历史
  clearHistory() {
    this.errorHistory = [];
  }

  // 获取特定类型的错误
  getErrorsByType(type: ErrorType): AppError[] {
    return this.errorHistory.filter((error) => error.type === type);
  }
}

// 全局错误处理器实例
export const globalErrorHandler = new ErrorHandler();

// 便捷的错误处理函数
export const handleError = (error: any, context?: string, config?: Partial<ErrorHandlerConfig>) => {
  return globalErrorHandler.handle(error, context, config);
};

// 特定类型的错误处理函数
export const handleNetworkError = (error: any) => {
  return handleError(error, 'network', { autoRetry: true, maxRetries: 3 });
};

export const handleAuthError = (error: any) => {
  return handleError(error, 'auth', { showToast: true, reportToService: true });
};

export const handleValidationError = (error: any) => {
  return handleError(error, 'validation', { showToast: true, logToConsole: false });
};

// 错误边界辅助函数
export const createErrorBoundary = (fallback: React.ComponentType<{ error: AppError }>) => {
  return class ErrorBoundary extends React.Component<
    { children: React.ReactNode },
    { error: AppError | null }
  > {
    constructor(props: { children: React.ReactNode }) {
      super(props);
      this.state = { error: null };
    }

    static getDerivedStateFromError(error: Error): { error: AppError } {
      return { error: ErrorClassifier.classify(error) };
    }

    componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
      handleError(error, 'react_boundary', { reportToService: true });
    }

    render() {
      if (this.state.error) {
        return React.createElement(fallback, { error: this.state.error });
      }

      return this.props.children;
    }
  };
};
