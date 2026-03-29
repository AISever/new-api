/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Input,
  Radio,
  RadioGroup,
  Space,
  Switch,
  Tag,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconArrowDown,
  IconArrowUp,
  IconDelete,
  IconPlus,
  IconUpload,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';
import {
  createDefaultHelpGroupCards,
  createEmptyHelpGroupCard,
  getHelpGroupCardsValidationError,
  readHelpGroupCardsOption,
  stringifyHelpGroupCardsOption,
  getDefaultHelpGroupCardImageURL,
} from './utils/helpGroupCards';
import { useIsMobile } from '../../hooks/common/useIsMobile';

const MAX_IMAGE_SIZE = 2 * 1024 * 1024;

const HelpGroupCardsSetting = ({ value, onSave, saving }) => {
  const { t } = useTranslation();
  const [mode, setMode] = useState('visual');
  const [cards, setCards] = useState(createDefaultHelpGroupCards());
  const [jsonValue, setJsonValue] = useState('[]');
  const [jsonError, setJsonError] = useState('');
  const [sourceError, setSourceError] = useState('');
  const [uploadingCardId, setUploadingCardId] = useState('');
  const fileInputRefs = useRef({});
  const isMobile = useIsMobile();

  const getJsonErrorMessage = (errorType) => {
    if (errorType === 'invalid_shape') {
      return t('群聊卡片 JSON 必须是数组');
    }
    return t('群聊卡片 JSON 格式无效');
  };

  const getCardValidationMessage = (error) => {
    if (!error) {
      return '';
    }
    if (error.type === 'missing_id') {
      return t('第 {{index}} 张群聊卡片缺少 id', {
        index: error.index + 1,
      });
    }
    if (error.type === 'missing_title') {
      return t('第 {{index}} 张群聊卡片缺少标题', {
        index: error.index + 1,
      });
    }
    if (error.type === 'duplicate_id') {
      return t('群聊卡片 id 重复：{{id}}', {
        id: error.id,
      });
    }
    return t('群聊卡片配置无效');
  };

  const syncCardsState = (nextCards) => {
    setCards(nextCards);
    setJsonValue(stringifyHelpGroupCardsOption(nextCards));
    setJsonError('');
  };

  useEffect(() => {
    const nextState = readHelpGroupCardsOption(
      value,
      createDefaultHelpGroupCards(),
    );
    setCards(nextState.cards);
    setJsonValue(
      nextState.error ? value || '[]' : stringifyHelpGroupCardsOption(nextState.cards),
    );
    const nextError = nextState.error ? getJsonErrorMessage(nextState.error) : '';
    setJsonError(nextError);
    setSourceError(nextError);
    if (nextError) {
      setMode('json');
    }
  }, [value]);

  const sortedCards = useMemo(
    () => [...cards].sort((left, right) => left.sort_order - right.sort_order),
    [cards],
  );

  const reindexCards = (nextCards) =>
    nextCards.map((card, index) => ({
      ...card,
      sort_order: index,
    }));

  const updateCard = (cardId, updates) => {
    const nextCards = cards.map((card) =>
      card.id === cardId ? { ...card, ...updates } : card,
    );
    syncCardsState(reindexCards(nextCards));
  };

  const handleAddCard = () => {
    syncCardsState(
      reindexCards([...cards, createEmptyHelpGroupCard(cards.length)]),
    );
  };

  const handleDeleteCard = (cardId) => {
    syncCardsState(reindexCards(cards.filter((card) => card.id !== cardId)));
  };

  const handleMoveCard = (cardId, direction) => {
    const currentIndex = sortedCards.findIndex((card) => card.id === cardId);
    const targetIndex = currentIndex + direction;
    if (currentIndex === -1 || targetIndex < 0 || targetIndex >= sortedCards.length) {
      return;
    }
    const nextCards = [...sortedCards];
    const [card] = nextCards.splice(currentIndex, 1);
    nextCards.splice(targetIndex, 0, card);
    syncCardsState(reindexCards(nextCards));
  };

  const handleJsonChange = (nextValue) => {
    setJsonValue(nextValue);
    const nextState = readHelpGroupCardsOption(
      nextValue,
      createDefaultHelpGroupCards(),
    );
    if (nextState.error) {
      const message = getJsonErrorMessage(nextState.error);
      setJsonError(message);
      setSourceError(message);
      return;
    }
    setCards(nextState.cards);
    setJsonError('');
    setSourceError('');
  };

  const handleSave = async () => {
    if (mode === 'json') {
      const nextState = readHelpGroupCardsOption(
        jsonValue,
        createDefaultHelpGroupCards(),
      );
      if (nextState.error) {
        const message = getJsonErrorMessage(nextState.error);
        setJsonError(message);
        setSourceError(message);
        showError(t('请先修正群聊卡片 JSON'));
        return;
      }
      const validationError = getHelpGroupCardsValidationError(nextState.cards);
      if (validationError) {
        const message = getCardValidationMessage(validationError);
        setJsonError(message);
        showError(message);
        return;
      }
      setCards(nextState.cards);
      setJsonError('');
      setSourceError('');
      await onSave(stringifyHelpGroupCardsOption(nextState.cards));
      return;
    }
    if (sourceError) {
      showError(t('当前群聊卡片 JSON 无效，请先切换到 JSON 模式修正'));
      return;
    }
    const validationError = getHelpGroupCardsValidationError(cards);
    if (validationError) {
      showError(getCardValidationMessage(validationError));
      return;
    }
    await onSave(stringifyHelpGroupCardsOption(cards));
  };

  const triggerFileSelect = (cardId) => {
    fileInputRefs.current[cardId]?.click();
  };

  const handleUploadImage = async (cardId, event) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) {
      return;
    }
    if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
      showError(t('仅支持 PNG、JPG、JPEG、WEBP 图片'));
      return;
    }
    if (file.size > MAX_IMAGE_SIZE) {
      showError(t('图片大小不能超过 2MB'));
      return;
    }

    const formData = new FormData();
    formData.append('category', 'help-group-card');
    formData.append('file', file);

    setUploadingCardId(cardId);
    try {
      const res = await API.post('/api/site-assets/upload', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      updateCard(cardId, { image_url: data.url });
      showSuccess(t('群聊卡片图片上传成功'));
    } catch (error) {
      showError(error?.message || t('群聊卡片图片上传失败'));
    } finally {
      setUploadingCardId('');
    }
  };

  return (
    <Card>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          gap: '12px',
          flexWrap: 'wrap',
          marginBottom: '16px',
        }}
      >
        <div>
          <Typography.Title heading={5} style={{ marginBottom: 4 }}>
            {t('群聊卡片')}
          </Typography.Title>
          <Typography.Text type='secondary'>
            {t('配置帮助页顶部展示的群聊卡片，支持多张卡片与图片上传。')}
          </Typography.Text>
        </div>
        <RadioGroup
          value={mode}
          type='button'
          onChange={(event) => setMode(event.target.value)}
        >
          <Radio value='visual'>{t('可视化')}</Radio>
          <Radio value='json'>JSON</Radio>
        </RadioGroup>
      </div>

      <Banner
        fullMode={false}
        type='info'
        closeIcon={null}
        description={t('图片将上传到服务器本地站点资源目录，建议使用不超过 2MB 的 PNG/JPG/WEBP。')}
        style={{ marginBottom: '16px' }}
      />
      {sourceError ? (
        <Banner
          fullMode={false}
          type='warning'
          closeIcon={null}
          description={t('检测到当前群聊卡片配置无效。请先在 JSON 模式下修正，再回到可视化模式编辑。')}
          style={{ marginBottom: '16px' }}
        />
      ) : null}

      {mode === 'visual' ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <Button
            icon={<IconPlus />}
            onClick={handleAddCard}
            disabled={Boolean(sourceError)}
          >
            {t('新增群聊卡片')}
          </Button>
          {sortedCards.length === 0 ? (
            <Typography.Text type='secondary'>
              {t('当前没有群聊卡片，点击上方按钮新增。')}
            </Typography.Text>
          ) : null}
          {sortedCards.map((card, index) => (
            <Card key={card.id} bodyStyle={{ padding: '16px' }}>
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: isMobile ? '1fr' : '220px minmax(0, 1fr)',
                  gap: '16px',
                }}
              >
                <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  <img
                    src={card.image_url || getDefaultHelpGroupCardImageURL()}
                    alt={card.title || t('群聊卡片图片')}
                    style={{
                      width: '100%',
                      aspectRatio: '1 / 1',
                      objectFit: 'cover',
                      borderRadius: '12px',
                      background: 'var(--semi-color-fill-0)',
                    }}
                  />
                  <Space wrap>
                    <Button
                      icon={<IconUpload />}
                      loading={uploadingCardId === card.id}
                      disabled={Boolean(sourceError)}
                      onClick={() => triggerFileSelect(card.id)}
                    >
                      {card.image_url ? t('重新上传图片') : t('上传图片')}
                    </Button>
                    <Button
                      disabled={Boolean(sourceError)}
                      onClick={() => updateCard(card.id, { image_url: '' })}
                    >
                      {t('清除图片')}
                    </Button>
                    <input
                      ref={(node) => {
                        fileInputRefs.current[card.id] = node;
                      }}
                      type='file'
                      accept='image/png,image/jpeg,image/webp'
                      style={{ display: 'none' }}
                      onChange={(event) => handleUploadImage(card.id, event)}
                    />
                  </Space>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      gap: '12px',
                      flexWrap: 'wrap',
                    }}
                  >
                    <Space wrap>
                      <Tag color='blue'>#{index + 1}</Tag>
                      <Switch
                        checked={card.enabled}
                        disabled={Boolean(sourceError)}
                        onChange={(checked) => updateCard(card.id, { enabled: checked })}
                        checkedText={t('显示')}
                        uncheckedText={t('隐藏')}
                      />
                    </Space>
                    <Space wrap>
                      <Button
                        icon={<IconArrowUp />}
                        disabled={Boolean(sourceError) || index === 0}
                        onClick={() => handleMoveCard(card.id, -1)}
                      />
                      <Button
                        icon={<IconArrowDown />}
                        disabled={
                          Boolean(sourceError) || index === sortedCards.length - 1
                        }
                        onClick={() => handleMoveCard(card.id, 1)}
                      />
                      <Button
                        type='danger'
                        icon={<IconDelete />}
                        disabled={Boolean(sourceError)}
                        onClick={() => handleDeleteCard(card.id)}
                      />
                    </Space>
                  </div>

                  <Input
                    value={card.label}
                    disabled={Boolean(sourceError)}
                    placeholder={t('标签，例如：群聊：AIGC 交流二群')}
                    onChange={(nextValue) => updateCard(card.id, { label: nextValue })}
                  />
                  <Input
                    value={card.title}
                    disabled={Boolean(sourceError)}
                    placeholder={t('标题')}
                    onChange={(nextValue) => updateCard(card.id, { title: nextValue })}
                  />
                  <TextArea
                    value={card.description}
                    disabled={Boolean(sourceError)}
                    placeholder={t('描述')}
                    autosize={{ minRows: 3, maxRows: 6 }}
                    onChange={(nextValue) => updateCard(card.id, { description: nextValue })}
                  />
                </div>
              </div>
            </Card>
          ))}
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          <TextArea
            value={jsonValue}
            onChange={handleJsonChange}
            autosize={{ minRows: 10, maxRows: 24 }}
            style={{ fontFamily: 'JetBrains Mono, Consolas' }}
          />
          {jsonError ? (
            <Typography.Text type='danger'>{jsonError}</Typography.Text>
          ) : null}
        </div>
      )}

      <div style={{ marginTop: '16px', display: 'flex', justifyContent: 'flex-end' }}>
        <Button type='primary' onClick={handleSave} loading={saving}>
          {t('保存群聊卡片设置')}
        </Button>
      </div>
    </Card>
  );
};

export default HelpGroupCardsSetting;
