'use client';

import { useState, useEffect } from 'react';
import { FileText, Search, Star, Clock, User, ChevronDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';

interface EmailTemplate {
  id: number;
  name: string;
  description?: string;
  subject: string;
  htmlBody: string;
  textBody: string;
  category: string;
  isBuiltIn: boolean;
  isShared: boolean;
  usageCount: number;
  variables: string[];
  createdAt: string;
  updatedAt: string;
}

interface TemplateSelectorProps {
  onTemplateSelect: (template: EmailTemplate) => void;
  selectedTemplateId?: number;
}

export function TemplateSelector({ onTemplateSelect, selectedTemplateId }: TemplateSelectorProps) {
  const [templates, setTemplates] = useState<EmailTemplate[]>([]);
  const [filteredTemplates, setFilteredTemplates] = useState<EmailTemplate[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [isLoading, setIsLoading] = useState(false);

  // 模拟模板数据
  const mockTemplates: EmailTemplate[] = [
    {
      id: 1,
      name: '会议邀请',
      description: '标准会议邀请模板',
      subject: '会议邀请：{{meetingTitle}}',
      htmlBody:
        '<p>您好 {{recipientName}}，</p><p>诚邀您参加 {{meetingTitle}} 会议。</p><p>时间：{{meetingTime}}</p><p>地点：{{meetingLocation}}</p>',
      textBody:
        '您好 {{recipientName}}，诚邀您参加 {{meetingTitle}} 会议。时间：{{meetingTime}} 地点：{{meetingLocation}}',
      category: '工作',
      isBuiltIn: true,
      isShared: false,
      usageCount: 25,
      variables: ['recipientName', 'meetingTitle', 'meetingTime', 'meetingLocation'],
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
    },
    {
      id: 2,
      name: '感谢信',
      description: '客户感谢信模板',
      subject: '感谢您的支持',
      htmlBody:
        '<p>亲爱的 {{customerName}}，</p><p>感谢您选择我们的服务。我们将继续为您提供优质的服务。</p>',
      textBody: '亲爱的 {{customerName}}，感谢您选择我们的服务。我们将继续为您提供优质的服务。',
      category: '客户服务',
      isBuiltIn: false,
      isShared: true,
      usageCount: 12,
      variables: ['customerName'],
      createdAt: '2024-01-02T00:00:00Z',
      updatedAt: '2024-01-02T00:00:00Z',
    },
    {
      id: 3,
      name: '生日祝福',
      description: '生日祝福邮件模板',
      subject: '生日快乐！{{recipientName}}',
      htmlBody:
        '<p>亲爱的 {{recipientName}}，</p><p>🎉 祝您生日快乐！愿您的每一天都充满快乐和幸福。</p>',
      textBody: '亲爱的 {{recipientName}}，祝您生日快乐！愿您的每一天都充满快乐和幸福。',
      category: '个人',
      isBuiltIn: true,
      isShared: false,
      usageCount: 8,
      variables: ['recipientName'],
      createdAt: '2024-01-03T00:00:00Z',
      updatedAt: '2024-01-03T00:00:00Z',
    },
  ];

  // 获取模板列表
  const fetchTemplates = async () => {
    setIsLoading(true);
    try {
      // TODO: 调用API获取模板
      // const response = await apiClient.getTemplates();
      // setTemplates(response.data);

      // 使用模拟数据
      await new Promise((resolve) => setTimeout(resolve, 500));
      setTemplates(mockTemplates);
      setFilteredTemplates(mockTemplates);
    } catch (error) {
      console.error('Failed to fetch templates:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // 过滤模板
  const filterTemplates = () => {
    let filtered = templates;

    // 按分类过滤
    if (selectedCategory !== 'all') {
      filtered = filtered.filter((template) => template.category === selectedCategory);
    }

    // 按搜索关键词过滤
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (template) =>
          template.name.toLowerCase().includes(query) ||
          template.description?.toLowerCase().includes(query) ||
          template.subject.toLowerCase().includes(query)
      );
    }

    setFilteredTemplates(filtered);
  };

  // 获取所有分类
  const getCategories = () => {
    const categories = Array.from(new Set(templates.map((t) => t.category)));
    return ['all', ...categories];
  };

  // 格式化使用次数
  const formatUsageCount = (count: number) => {
    if (count === 0) return '未使用';
    if (count === 1) return '使用过 1 次';
    return `使用过 ${count} 次`;
  };

  useEffect(() => {
    fetchTemplates();
  }, []);

  useEffect(() => {
    filterTemplates();
  }, [templates, searchQuery, selectedCategory]);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" className="gap-2">
          <FileText className="w-4 h-4" />
          选择模板
          <ChevronDown className="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>

      <DropdownMenuContent className="w-96 p-0">
        {/* 搜索和过滤 */}
        <div className="p-3 border-b">
          <div className="space-y-2">
            {/* 搜索框 */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
              <Input
                placeholder="搜索模板..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 h-8"
              />
            </div>

            {/* 分类选择 */}
            <div className="flex gap-1 flex-wrap">
              {getCategories().map((category) => (
                <Button
                  key={category}
                  variant={selectedCategory === category ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setSelectedCategory(category)}
                  className="h-6 px-2 text-xs"
                >
                  {category === 'all' ? '全部' : category}
                </Button>
              ))}
            </div>
          </div>
        </div>

        {/* 模板列表 */}
        <ScrollArea className="max-h-80">
          {isLoading ? (
            <div className="p-4 text-center text-gray-500">加载中...</div>
          ) : filteredTemplates.length === 0 ? (
            <div className="p-4 text-center text-gray-500">
              {searchQuery || selectedCategory !== 'all' ? '没有找到匹配的模板' : '暂无模板'}
            </div>
          ) : (
            <div className="p-1">
              {filteredTemplates.map((template) => (
                <DropdownMenuItem
                  key={template.id}
                  onClick={() => onTemplateSelect(template)}
                  className={`p-3 cursor-pointer ${
                    selectedTemplateId === template.id ? 'bg-blue-50 dark:bg-blue-900/20' : ''
                  }`}
                >
                  <div className="w-full">
                    {/* 模板标题和标签 */}
                    <div className="flex items-start justify-between mb-1">
                      <div className="flex items-center gap-2">
                        <h4 className="font-medium text-sm text-gray-900 dark:text-gray-100">
                          {template.name}
                        </h4>
                        <div className="flex gap-1">
                          {template.isBuiltIn && (
                            <Badge variant="secondary" className="text-xs">
                              内置
                            </Badge>
                          )}
                          {template.isShared && (
                            <Badge variant="outline" className="text-xs">
                              共享
                            </Badge>
                          )}
                        </div>
                      </div>
                      {template.usageCount > 0 && <Star className="w-3 h-3 text-yellow-500" />}
                    </div>

                    {/* 模板描述 */}
                    {template.description && (
                      <p className="text-xs text-gray-600 dark:text-gray-400 mb-1">
                        {template.description}
                      </p>
                    )}

                    {/* 模板主题预览 */}
                    <p className="text-xs text-gray-500 dark:text-gray-400 mb-2 truncate">
                      主题: {template.subject}
                    </p>

                    {/* 模板信息 */}
                    <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
                      <div className="flex items-center gap-3">
                        <span className="flex items-center gap-1">
                          <User className="w-3 h-3" />
                          {template.category}
                        </span>
                        <span className="flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          {formatUsageCount(template.usageCount)}
                        </span>
                      </div>

                      {template.variables.length > 0 && (
                        <span className="text-blue-600 dark:text-blue-400">
                          {template.variables.length} 个变量
                        </span>
                      )}
                    </div>
                  </div>
                </DropdownMenuItem>
              ))}
            </div>
          )}
        </ScrollArea>

        {/* 底部操作 */}
        <div className="p-3 border-t">
          <Button variant="ghost" size="sm" className="w-full text-xs">
            管理模板
          </Button>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
