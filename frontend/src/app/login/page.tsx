'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent } from '@/components/ui/card';
import { useAuth } from '@/hooks/use-auth';
import { AuthRoute } from '@/components/auth/route-guard';

const loginSchema = z.object({
  username: z.string().min(1, '请输入用户名'),
  password: z.string().min(1, '请输入密码'),
});

type LoginForm = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const { login, isLoggingIn } = useAuth();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
  });

  const onSubmit = (data: LoginForm) => {
    login(data);
  };

  return (
    <AuthRoute>
      <div className="min-h-screen flex items-center justify-center bg-white dark:bg-gray-900 py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-md w-full space-y-16">
          {/* 艺术字标题 */}
          <div className="text-center">
            <h1 className="text-8xl font-light text-gray-900 dark:text-gray-100 mb-4 tracking-wider">
              花火邮箱
            </h1>
            <div className="w-16 h-px bg-gray-400 dark:bg-gray-600 mx-auto"></div>
          </div>

          {/* 登录卡片 */}
          <Card className="border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-sm">
            <CardContent className="pt-8 pb-8 px-8">
              <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
                <div className="space-y-4">
                  <Input
                    id="username"
                    type="text"
                    placeholder="用户名"
                    {...register('username')}
                    className={`h-12 border-0 border-b border-gray-300 dark:border-gray-600 rounded-none bg-transparent focus:border-gray-900 dark:focus:border-gray-100 focus:ring-0 text-gray-900 dark:text-gray-100 placeholder:text-gray-500 dark:placeholder:text-gray-400 transition-colors ${
                      errors.username ? 'border-red-500 focus:border-red-500' : ''
                    }`}
                  />
                  {errors.username && (
                    <p className="text-sm text-red-500 mt-1">{errors.username.message}</p>
                  )}
                </div>

                <div className="space-y-4">
                  <Input
                    id="password"
                    type="password"
                    placeholder="密码"
                    {...register('password')}
                    className={`h-12 border-0 border-b border-gray-300 dark:border-gray-600 rounded-none bg-transparent focus:border-gray-900 dark:focus:border-gray-100 focus:ring-0 text-gray-900 dark:text-gray-100 placeholder:text-gray-500 dark:placeholder:text-gray-400 transition-colors ${
                      errors.password ? 'border-red-500 focus:border-red-500' : ''
                    }`}
                  />
                  {errors.password && (
                    <p className="text-sm text-red-500 mt-1">{errors.password.message}</p>
                  )}
                </div>

                <div className="pt-4">
                  <Button
                    type="submit"
                    className="w-full h-12 bg-gray-900 dark:bg-gray-100 hover:bg-gray-800 dark:hover:bg-gray-200 text-white dark:text-gray-900 font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    disabled={isLoggingIn}
                  >
                    {isLoggingIn ? (
                      <div className="flex items-center gap-2">
                        <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
                        登录中
                      </div>
                    ) : (
                      '登录'
                    )}
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </div>
      </div>
    </AuthRoute>
  );
}
