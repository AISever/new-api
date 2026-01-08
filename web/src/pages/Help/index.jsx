/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useState, useContext } from 'react';
import { Tabs, TabPane, Card, Typography, Collapse, Tag, Banner, Toast } from '@douyinfe/semi-ui';
import { IconCode, IconTerminal, IconDesktop, IconApps, IconCopy } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';

const { Title, Paragraph, Text } = Typography;

const Help = () => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const [activeTab, setActiveTab] = useState('claude-code');
  const [activePlatform, setActivePlatform] = useState('windows');
  const actualTheme = useActualTheme();

  // 获取服务器地址
  const serverAddress = statusState?.status?.server_address || window.location.origin;

  const CodeBlock = ({ children }) => {
    const isDark = actualTheme === 'dark';
    
    const handleCopy = () => {
      navigator.clipboard.writeText(children).then(() => {
        Toast.success(t('已复制到剪贴板'));
      }).catch(() => {
        Toast.error(t('复制失败'));
      });
    };

    return (
      <div className="relative my-2 group">
        <pre 
          style={{
            backgroundColor: isDark ? '#1e1e1e' : '#f5f5f5',
            color: isDark ? '#d4d4d4' : '#333333',
            padding: '16px',
            borderRadius: '8px',
            overflow: 'auto',
            fontSize: '13px',
            lineHeight: '1.5',
            border: isDark ? '1px solid #333' : '1px solid #e0e0e0',
          }}
        >
          <code>{children}</code>
        </pre>
        <button
          onClick={handleCopy}
          style={{
            position: 'absolute',
            top: '8px',
            right: '8px',
            padding: '4px 8px',
            backgroundColor: isDark ? '#333' : '#e0e0e0',
            color: isDark ? '#d4d4d4' : '#333',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer',
            opacity: 0.7,
            transition: 'opacity 0.2s',
            display: 'flex',
            alignItems: 'center',
            gap: '4px',
            fontSize: '12px',
          }}
          onMouseEnter={(e) => e.target.style.opacity = 1}
          onMouseLeave={(e) => e.target.style.opacity = 0.7}
        >
          <IconCopy size="small" />
          {t('复制')}
        </button>
      </div>
    );
  };

  const StepCard = ({ step, title, children }) => (
    <Card className="mb-4" title={<span><Tag color="blue">{step}</Tag> {title}</span>}>
      {children}
    </Card>
  );

  // Claude Code 教程
  const ClaudeCodeTutorial = () => (
    <div>
      <Banner
        type="info"
        description={t('Claude Code 是 Anthropic 官方推出的 AI 编程助手，支持代码生成、代码解释、代码优化等功能。')}
        className="mb-4"
      />
      
      <Tabs activeKey={activePlatform} onChange={setActivePlatform}>
        <TabPane tab={<span><IconDesktop /> Windows</span>} itemKey="windows">
          <StepCard step="1" title={t('安装 Node.js 环境')}>
            <Paragraph>{t('Claude Code 需要 Node.js 环境才能运行（版本需大于 18）。')}</Paragraph>
            <Title heading={6}>{t('方法一：官网下载（推荐）')}</Title>
            <Paragraph>
              1. {t('打开浏览器访问')} <a href="https://nodejs.org/" target="_blank" rel="noopener noreferrer">https://nodejs.org/</a><br/>
              2. {t('点击 "LTS" 版本进行下载')}<br/>
              3. {t('下载完成后双击 .msi 文件，按照安装向导完成安装')}
            </Paragraph>
            <Title heading={6}>{t('方法二：使用包管理器')}</Title>
            <CodeBlock>{`# 使用 Chocolatey
choco install nodejs
# 或使用 Scoop
scoop install nodejs`}</CodeBlock>
            <Title heading={6}>{t('验证安装')}</Title>
            <CodeBlock>{`node --version
npm --version`}</CodeBlock>
          </StepCard>

          <StepCard step="2" title={t('安装 Claude Code')}>
            <Paragraph>{t('打开 PowerShell 或 CMD，运行以下命令：')}</Paragraph>
            <CodeBlock>{`npm install -g @anthropic-ai/claude-code --registry=https://registry.npmmirror.com`}</CodeBlock>
            <Title heading={6}>{t('验证安装')}</Title>
            <CodeBlock>{`claude --version`}</CodeBlock>
          </StepCard>

          <StepCard step="3" title={t('配置环境变量')}>
            <Title heading={6}>{t('方法一：通过配置文件设置（推荐）')}</Title>
            <Paragraph>{t('编辑文件')} <Text code>C:\\Users\\你的用户名\\.claude\\settings.json</Text></Paragraph>
            <CodeBlock language="json">{`{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "你的API密钥",
    "ANTHROPIC_BASE_URL": "${serverAddress}/api",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"
  },
  "permissions": {
    "allow": [],
    "deny": []
  }
}`}</CodeBlock>

            <Title heading={6}>{t('方法二：PowerShell 临时设置')}</Title>
            <CodeBlock>{`$env:ANTHROPIC_BASE_URL = "${serverAddress}/api"
$env:ANTHROPIC_AUTH_TOKEN = "你的API密钥"`}</CodeBlock>

            <Title heading={6}>{t('方法三：PowerShell 永久设置')}</Title>
            <CodeBlock>{`[System.Environment]::SetEnvironmentVariable("ANTHROPIC_BASE_URL", "${serverAddress}/api", [System.EnvironmentVariableTarget]::User)
[System.Environment]::SetEnvironmentVariable("ANTHROPIC_AUTH_TOKEN", "你的API密钥", [System.EnvironmentVariableTarget]::User)`}</CodeBlock>
          </StepCard>

          <StepCard step="4" title={t('开始使用')}>
            <CodeBlock>{`# 启动 Claude Code
claude

# 在特定项目中使用
cd C:\\path\\to\\your\\project
claude`}</CodeBlock>
          </StepCard>
        </TabPane>

        <TabPane tab={<span><IconTerminal /> macOS</span>} itemKey="macos">
          <StepCard step="1" title={t('安装 Node.js 环境')}>
            <Title heading={6}>{t('方法一：使用 Homebrew（推荐）')}</Title>
            <CodeBlock>{`# 更新 Homebrew
brew update
# 安装 Node.js
brew install node`}</CodeBlock>
            <Title heading={6}>{t('方法二：官网下载')}</Title>
            <Paragraph>
              {t('访问')} <a href="https://nodejs.org/" target="_blank" rel="noopener noreferrer">https://nodejs.org/</a> {t('下载 macOS 版本')}
            </Paragraph>
            <Title heading={6}>{t('验证安装')}</Title>
            <CodeBlock>{`node --version
npm --version`}</CodeBlock>
          </StepCard>

          <StepCard step="2" title={t('安装 Claude Code')}>
            <CodeBlock>{`npm install -g @anthropic-ai/claude-code`}</CodeBlock>
          </StepCard>

          <StepCard step="3" title={t('配置环境变量')}>
            <Title heading={6}>{t('临时设置')}</Title>
            <CodeBlock>{`export ANTHROPIC_BASE_URL="${serverAddress}/api"
export ANTHROPIC_AUTH_TOKEN="你的API密钥"`}</CodeBlock>
            <Title heading={6}>{t('永久设置（zsh）')}</Title>
            <CodeBlock>{`echo 'export ANTHROPIC_BASE_URL="${serverAddress}/api"' >> ~/.zshrc
echo 'export ANTHROPIC_AUTH_TOKEN="你的API密钥"' >> ~/.zshrc
source ~/.zshrc`}</CodeBlock>
          </StepCard>

          <StepCard step="4" title={t('开始使用')}>
            <CodeBlock>{`claude`}</CodeBlock>
          </StepCard>
        </TabPane>

        <TabPane tab={<span><IconTerminal /> Linux</span>} itemKey="linux">
          <StepCard step="1" title={t('安装 Node.js 环境')}>
            <Title heading={6}>{t('方法一：使用官方仓库（推荐）')}</Title>
            <CodeBlock>{`# 添加 NodeSource 仓库
curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
# 安装 Node.js
sudo apt-get install -y nodejs`}</CodeBlock>
            <Title heading={6}>{t('方法二：使用系统包管理器')}</Title>
            <CodeBlock>{`# Ubuntu/Debian
sudo apt update
sudo apt install nodejs npm

# CentOS/RHEL/Fedora
sudo dnf install nodejs npm`}</CodeBlock>
          </StepCard>

          <StepCard step="2" title={t('安装 Claude Code')}>
            <CodeBlock>{`npm install -g @anthropic-ai/claude-code
# 如果遇到权限问题
sudo npm install -g @anthropic-ai/claude-code`}</CodeBlock>
          </StepCard>

          <StepCard step="3" title={t('配置环境变量')}>
            <Title heading={6}>{t('永久设置（bash）')}</Title>
            <CodeBlock>{`echo 'export ANTHROPIC_BASE_URL="${serverAddress}/api"' >> ~/.bashrc
echo 'export ANTHROPIC_AUTH_TOKEN="你的API密钥"' >> ~/.bashrc
source ~/.bashrc`}</CodeBlock>
          </StepCard>

          <StepCard step="4" title={t('开始使用')}>
            <CodeBlock>{`claude`}</CodeBlock>
          </StepCard>
        </TabPane>
      </Tabs>
    </div>
  );

  // Codex 教程
  const CodexTutorial = () => (
    <div>
      <Banner
        type="info"
        description={t('Codex 是 OpenAI 推出的 AI 编程助手，基于 GPT 模型，支持多种编程语言。')}
        className="mb-4"
      />
      
      <Tabs activeKey={activePlatform} onChange={setActivePlatform}>
        <TabPane tab={<span><IconDesktop /> Windows</span>} itemKey="windows">
          <StepCard step="1" title={t('安装 Node.js 环境')}>
            <Paragraph>{t('参考 Claude Code 的 Node.js 安装步骤。')}</Paragraph>
          </StepCard>

          <StepCard step="2" title={t('安装 Codex')}>
            <CodeBlock>{`npm install -g @openai/codex`}</CodeBlock>
            <Title heading={6}>{t('验证安装')}</Title>
            <CodeBlock>{`codex --version`}</CodeBlock>
          </StepCard>

          <StepCard step="3" title={t('配置环境变量')}>
            <Title heading={6}>{t('方法一：配置文件设置')}</Title>
            <Paragraph>{t('在')} <Text code>C:\\Users\\你的用户名\\.codex</Text> {t('目录下创建')} <Text code>config.toml</Text></Paragraph>
            
            <Title heading={6}>{t('方法二：PowerShell 设置')}</Title>
            <CodeBlock>{`$env:OPENAI_BASE_URL = "${serverAddress}/v1"
$env:OPENAI_API_KEY = "你的API密钥"`}</CodeBlock>

            <Title heading={6}>{t('永久设置')}</Title>
            <CodeBlock>{`[System.Environment]::SetEnvironmentVariable("OPENAI_BASE_URL", "${serverAddress}/v1", [System.EnvironmentVariableTarget]::User)
[System.Environment]::SetEnvironmentVariable("OPENAI_API_KEY", "你的API密钥", [System.EnvironmentVariableTarget]::User)`}</CodeBlock>
          </StepCard>

          <StepCard step="4" title={t('开始使用')}>
            <CodeBlock>{`codex`}</CodeBlock>
          </StepCard>
        </TabPane>

        <TabPane tab={<span><IconTerminal /> macOS</span>} itemKey="macos">
          <StepCard step="1" title={t('安装 Node.js 环境')}>
            <CodeBlock>{`brew update
brew install node`}</CodeBlock>
          </StepCard>

          <StepCard step="2" title={t('安装 Codex')}>
            <CodeBlock>{`npm install -g @openai/codex`}</CodeBlock>
          </StepCard>

          <StepCard step="3" title={t('配置环境变量')}>
            <Title heading={6}>{t('永久设置（zsh）')}</Title>
            <CodeBlock>{`echo 'export OPENAI_BASE_URL="${serverAddress}/v1"' >> ~/.zshrc
echo 'export OPENAI_API_KEY="你的API密钥"' >> ~/.zshrc
source ~/.zshrc`}</CodeBlock>
          </StepCard>

          <StepCard step="4" title={t('开始使用')}>
            <CodeBlock>{`codex`}</CodeBlock>
          </StepCard>
        </TabPane>

        <TabPane tab={<span><IconTerminal /> Linux</span>} itemKey="linux">
          <StepCard step="1" title={t('安装 Node.js 环境')}>
            <CodeBlock>{`curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
sudo apt-get install -y nodejs`}</CodeBlock>
          </StepCard>

          <StepCard step="2" title={t('安装 Codex')}>
            <CodeBlock>{`npm install -g @openai/codex`}</CodeBlock>
          </StepCard>

          <StepCard step="3" title={t('配置环境变量')}>
            <CodeBlock>{`echo 'export OPENAI_BASE_URL="${serverAddress}/v1"' >> ~/.bashrc
echo 'export OPENAI_API_KEY="你的API密钥"' >> ~/.bashrc
source ~/.bashrc`}</CodeBlock>
          </StepCard>

          <StepCard step="4" title={t('开始使用')}>
            <CodeBlock>{`codex`}</CodeBlock>
          </StepCard>
        </TabPane>
      </Tabs>
    </div>
  );

  // Gemini CLI 教程
  const GeminiTutorial = () => (
    <div>
      <Banner
        type="info"
        description={t('Gemini CLI 是 Google 推出的 AI 编程助手，基于 Gemini 模型。')}
        className="mb-4"
      />
      
      <Tabs activeKey={activePlatform} onChange={setActivePlatform}>
        <TabPane tab={<span><IconDesktop /> Windows</span>} itemKey="windows">
          <StepCard step="1" title={t('安装 Node.js 环境')}>
            <Paragraph>{t('参考 Claude Code 的 Node.js 安装步骤。')}</Paragraph>
          </StepCard>

          <StepCard step="2" title={t('安装 Claude Code（Gemini CLI 依赖）')}>
            <CodeBlock>{`npm install -g @anthropic-ai/claude-code`}</CodeBlock>
          </StepCard>

          <StepCard step="3" title={t('配置 Gemini CLI 环境变量')}>
            <Title heading={6}>{t('PowerShell 临时设置')}</Title>
            <CodeBlock>{`$env:CODE_ASSIST_ENDPOINT = "${serverAddress}/gemini"
$env:GOOGLE_CLOUD_ACCESS_TOKEN = "你的API密钥"
$env:GOOGLE_GENAI_USE_GCA = "true"`}</CodeBlock>

            <Title heading={6}>{t('PowerShell 永久设置')}</Title>
            <CodeBlock>{`[System.Environment]::SetEnvironmentVariable("CODE_ASSIST_ENDPOINT", "${serverAddress}/gemini", [System.EnvironmentVariableTarget]::User)
[System.Environment]::SetEnvironmentVariable("GOOGLE_CLOUD_ACCESS_TOKEN", "你的API密钥", [System.EnvironmentVariableTarget]::User)
[System.Environment]::SetEnvironmentVariable("GOOGLE_GENAI_USE_GCA", "true", [System.EnvironmentVariableTarget]::User)`}</CodeBlock>
          </StepCard>

          <StepCard step="4" title={t('验证配置')}>
            <CodeBlock>{`echo $env:CODE_ASSIST_ENDPOINT
echo $env:GOOGLE_CLOUD_ACCESS_TOKEN
echo $env:GOOGLE_GENAI_USE_GCA`}</CodeBlock>
          </StepCard>
        </TabPane>

        <TabPane tab={<span><IconTerminal /> macOS</span>} itemKey="macos">
          <StepCard step="1" title={t('安装依赖')}>
            <CodeBlock>{`brew update
brew install node
npm install -g @anthropic-ai/claude-code`}</CodeBlock>
          </StepCard>

          <StepCard step="2" title={t('配置环境变量')}>
            <Title heading={6}>{t('永久设置（zsh）')}</Title>
            <CodeBlock>{`echo 'export CODE_ASSIST_ENDPOINT="${serverAddress}/gemini"' >> ~/.zshrc
echo 'export GOOGLE_CLOUD_ACCESS_TOKEN="你的API密钥"' >> ~/.zshrc
echo 'export GOOGLE_GENAI_USE_GCA="true"' >> ~/.zshrc
source ~/.zshrc`}</CodeBlock>
          </StepCard>

          <StepCard step="3" title={t('验证配置')}>
            <CodeBlock>{`echo $CODE_ASSIST_ENDPOINT
echo $GOOGLE_CLOUD_ACCESS_TOKEN
echo $GOOGLE_GENAI_USE_GCA`}</CodeBlock>
          </StepCard>
        </TabPane>

        <TabPane tab={<span><IconTerminal /> Linux</span>} itemKey="linux">
          <StepCard step="1" title={t('安装依赖')}>
            <CodeBlock>{`curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
sudo apt-get install -y nodejs
npm install -g @anthropic-ai/claude-code`}</CodeBlock>
          </StepCard>

          <StepCard step="2" title={t('配置环境变量')}>
            <CodeBlock>{`echo 'export CODE_ASSIST_ENDPOINT="${serverAddress}/gemini"' >> ~/.bashrc
echo 'export GOOGLE_CLOUD_ACCESS_TOKEN="你的API密钥"' >> ~/.bashrc
echo 'export GOOGLE_GENAI_USE_GCA="true"' >> ~/.bashrc
source ~/.bashrc`}</CodeBlock>
          </StepCard>

          <StepCard step="3" title={t('验证配置')}>
            <CodeBlock>{`echo $CODE_ASSIST_ENDPOINT
echo $GOOGLE_CLOUD_ACCESS_TOKEN
echo $GOOGLE_GENAI_USE_GCA`}</CodeBlock>
          </StepCard>
        </TabPane>
      </Tabs>
    </div>
  );

  return (
    <div className="mt-[60px] px-4 py-6 max-w-6xl mx-auto">
      {/* 微信群邀请 */}
      <Card className="mb-6" bodyStyle={{ padding: '16px 24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '20px' }}>
          <img 
            src="/WeChat-qun.jpg" 
            alt={t('微信群二维码')}
            style={{ 
              width: '180px', 
              height: '180px', 
              borderRadius: '8px',
              flexShrink: 0
            }} 
          />
          <div>
            <Title heading={5} style={{ marginBottom: '8px' }}>{t('欢迎加微信群沟通交流')}</Title>
            <Paragraph style={{ marginBottom: 0, color: 'var(--semi-color-text-2)' }}>
              {t('扫描二维码加入 AIGC 交流群，与更多开发者一起探讨 AI 编程工具的使用技巧和最佳实践。')}
            </Paragraph>
          </div>
        </div>
      </Card>

      <Title heading={2} className="mb-6">{t('AI 编程工具使用指南')}</Title>
      
      <Banner
        type="success"
        description={
          <span>
            {t('当前服务地址：')} <Text code copyable>{serverAddress}</Text>
            {t('，请在下方教程中使用此地址配置您的 API 端点。')}
          </span>
        }
        className="mb-6"
      />

      <Tabs activeKey={activeTab} onChange={setActiveTab} type="card">
        <TabPane 
          tab={<span><IconCode /> Claude Code</span>} 
          itemKey="claude-code"
        >
          <ClaudeCodeTutorial />
        </TabPane>
        <TabPane 
          tab={<span><IconApps /> Codex</span>} 
          itemKey="codex"
        >
          <CodexTutorial />
        </TabPane>
        <TabPane 
          tab={<span><IconTerminal /> Gemini CLI</span>} 
          itemKey="gemini"
        >
          <GeminiTutorial />
        </TabPane>
      </Tabs>

      <Card className="mt-6" title={t('常见问题')}>
        <Collapse>
          <Collapse.Panel header={t('安装时提示权限错误')} itemKey="1">
            <Paragraph>
              {t('尝试以下解决方法：')}<br/>
              1. {t('使用管理员权限运行终端')}<br/>
              2. {t('使用 sudo 命令（Linux/macOS）')}<br/>
              3. {t('配置 npm 使用用户目录：')}
            </Paragraph>
            <CodeBlock>{`npm config set prefix ~/.npm-global
echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc
source ~/.bashrc`}</CodeBlock>
          </Collapse.Panel>
          <Collapse.Panel header={t('环境变量设置后不生效')} itemKey="2">
            <Paragraph>
              1. {t('确认修改了正确的配置文件')}<br/>
              2. {t('重新启动终端或运行 source 命令')}<br/>
              3. {t('Windows 用户需要重新打开 PowerShell 窗口')}
            </Paragraph>
          </Collapse.Panel>
          <Collapse.Panel header={t('API 连接失败')} itemKey="3">
            <Paragraph>
              1. {t('检查 API 密钥是否正确')}<br/>
              2. {t('检查服务地址是否正确')}<br/>
              3. {t('检查网络连接是否正常')}<br/>
              4. {t('确认账户余额充足')}
            </Paragraph>
          </Collapse.Panel>
          <Collapse.Panel header={t('Node.js 版本过低')} itemKey="4">
            <Paragraph>
              {t('需要 Node.js 18 或更高版本，请更新 Node.js：')}
            </Paragraph>
            <CodeBlock>{`# macOS
brew upgrade node

# Linux
curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
sudo apt-get install -y nodejs`}</CodeBlock>
          </Collapse.Panel>
        </Collapse>
      </Card>
    </div>
  );
};

export default Help;
