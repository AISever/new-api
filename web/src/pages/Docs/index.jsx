import React, { useContext, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Card,
  Empty,
  Skeleton,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { StatusContext } from '../../context/Status';
import { getDocsAvailability, showError } from '../../helpers';
import MarkdownRenderer from '../../components/common/markdown/MarkdownRenderer';
import { useIsMobile } from '../../hooks/common/useIsMobile';

const { Title, Paragraph, Text } = Typography;

function buildSidebarGroups(items) {
  const groups = new Map();

  items.forEach((item) => {
    const section = item.section || 'Docs';
    if (!groups.has(section)) {
      groups.set(section, []);
    }
    groups.get(section).push(item);
  });

  return Array.from(groups.entries()).map(([section, docs]) => ({
    section,
    docs,
  }));
}

export default function DocsPage() {
  const { docId } = useParams();
  const navigate = useNavigate();
  const [statusState] = useContext(StatusContext);
  const isMobile = useIsMobile();
  const [manifest, setManifest] = useState(null);
  const [markdown, setMarkdown] = useState('');
  const [manifestLoading, setManifestLoading] = useState(false);
  const [markdownLoading, setMarkdownLoading] = useState(false);

  const docsAvailability = useMemo(
    () => getDocsAvailability(statusState?.status || {}),
    [statusState?.status],
  );

  useEffect(() => {
    if (!docsAvailability.hasLocalDocs) {
      return;
    }

    let cancelled = false;
    const loadManifest = async () => {
      try {
        setManifestLoading(true);
        const response = await fetch(docsAvailability.manifestPath, {
          cache: 'no-store',
        });
        if (!response.ok) {
          throw new Error(`manifest ${response.status}`);
        }
        const data = await response.json();
        if (!cancelled) {
          setManifest(data);
        }
      } catch (error) {
        if (!cancelled) {
          showError('加载企业文档目录失败');
        }
      } finally {
        if (!cancelled) {
          setManifestLoading(false);
        }
      }
    };

    loadManifest();
    return () => {
      cancelled = true;
    };
  }, [docsAvailability.hasLocalDocs, docsAvailability.manifestPath]);

  useEffect(() => {
    if (!manifest?.items?.length) {
      return;
    }
    if (!docId) {
      navigate(`/docs/${manifest.items[0].id}`, { replace: true });
    }
  }, [docId, manifest, navigate]);

  const currentDoc = useMemo(() => {
    if (!manifest?.items?.length || !docId) {
      return null;
    }
    return manifest.items.find((item) => item.id === docId) || null;
  }, [docId, manifest]);

  useEffect(() => {
    if (!currentDoc?.markdownPath) {
      return;
    }

    let cancelled = false;
    const loadMarkdown = async () => {
      try {
        setMarkdownLoading(true);
        const response = await fetch(currentDoc.markdownPath, {
          cache: 'no-store',
        });
        if (!response.ok) {
          throw new Error(`markdown ${response.status}`);
        }
        const text = await response.text();
        if (!cancelled) {
          setMarkdown(text);
        }
      } catch (error) {
        if (!cancelled) {
          setMarkdown('');
          showError('加载企业文档内容失败');
        }
      } finally {
        if (!cancelled) {
          setMarkdownLoading(false);
        }
      }
    };

    loadMarkdown();
    return () => {
      cancelled = true;
    };
  }, [currentDoc]);

  const sidebarGroups = useMemo(
    () => buildSidebarGroups(manifest?.items || []),
    [manifest],
  );

  if (!docsAvailability.hasLocalDocs) {
    return (
      <div className='mt-[60px] px-4 py-6 max-w-5xl mx-auto'>
        <Empty
          title='未启用企业文档'
          description='当前环境没有配置本地文档清单，因此无法进入企业版文档中心。'
        />
      </div>
    );
  }

  return (
    <div className='mt-[60px] px-4 py-6 max-w-[1400px] mx-auto'>
      <div className='mb-6 flex flex-col gap-3'>
        <Tag color='green' style={{ width: 'fit-content' }}>
          企业版本地文档
        </Tag>
        <Title heading={2} style={{ margin: 0 }}>
          企业 API 文档中心
        </Title>
        <Paragraph style={{ margin: 0, color: 'var(--semi-color-text-2)' }}>
          当前文档由企业版自身托管，已与上游公开文档内容同步，并替换为企业环境的访问地址。
        </Paragraph>
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: isMobile ? '1fr' : '320px minmax(0, 1fr)',
          gap: '20px',
          alignItems: 'start',
        }}
      >
        <Card bodyStyle={{ padding: '16px' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            {manifestLoading ? (
              <Skeleton placeholder={<Skeleton.Paragraph rows={8} />} loading />
            ) : (
              sidebarGroups.map((group) => (
                <div key={group.section}>
                  <Text
                    strong
                    style={{
                      display: 'block',
                      marginBottom: '10px',
                    }}
                  >
                    {group.section}
                  </Text>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                    {group.docs.map((doc) => (
                      <Button
                        key={doc.id}
                        theme={doc.id === currentDoc?.id ? 'solid' : 'borderless'}
                        type={doc.id === currentDoc?.id ? 'primary' : 'tertiary'}
                        style={{
                          justifyContent: 'flex-start',
                          height: 'auto',
                          padding: '10px 12px',
                        }}
                        onClick={() => navigate(`/docs/${doc.id}`)}
                      >
                        <div style={{ textAlign: 'left' }}>
                          <div>{doc.title}</div>
                          {doc.hierarchy?.length ? (
                            <div
                              style={{
                                fontSize: '12px',
                                color: 'var(--semi-color-text-2)',
                                marginTop: '4px',
                                whiteSpace: 'normal',
                              }}
                            >
                              {doc.hierarchy.join(' / ')}
                            </div>
                          ) : null}
                        </div>
                      </Button>
                    ))}
                  </div>
                </div>
              ))
            )}
          </div>
        </Card>

        <Card bodyStyle={{ padding: '24px' }}>
          {currentDoc ? (
            <>
              <div style={{ marginBottom: '16px' }}>
                <Title heading={3} style={{ margin: 0 }}>
                  {currentDoc.title}
                </Title>
                {currentDoc.hierarchy?.length ? (
                  <Text type='tertiary'>
                    {currentDoc.hierarchy.join(' / ')}
                  </Text>
                ) : null}
              </div>
              {markdownLoading ? (
                <Skeleton
                  loading
                  placeholder={<Skeleton.Paragraph rows={14} />}
                />
              ) : (
                <MarkdownRenderer content={markdown} />
              )}
            </>
          ) : (
            <Empty
              title='请选择文档'
              description='从左侧目录选择一篇文档以开始阅读。'
            />
          )}
        </Card>
      </div>
    </div>
  );
}
