/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Input,
  Modal,
  Select,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';

const { Title, Text } = Typography;

const SeedanceAssets = () => {
  const { t } = useTranslation();
  const [groups, setGroups] = useState([]);
  const [assets, setAssets] = useState([]);
  const [activeGroupId, setActiveGroupId] = useState(null);
  const [keyword, setKeyword] = useState('');
  const [loading, setLoading] = useState(false);
  const [groupModalOpen, setGroupModalOpen] = useState(false);
  const [assetModalOpen, setAssetModalOpen] = useState(false);
  const [groupName, setGroupName] = useState('');
  const [assetName, setAssetName] = useState('');
  const [assetType, setAssetType] = useState('Image');
  const [assetUrl, setAssetUrl] = useState('');

  const loadGroups = useCallback(async () => {
    const res = await API.get('/api/seedance/groups?p=1&size=50');
    if (!res.data.success) {
      showError(res.data.message);
      return;
    }
    const items = res.data.data.items || [];
    setGroups(items);
    if (!activeGroupId && items.length > 0) {
      const defaultGroup = items.find((item) => item.is_default) || items[0];
      setActiveGroupId(defaultGroup.id);
    }
  }, [activeGroupId]);

  const loadAssets = useCallback(async () => {
    if (!activeGroupId) {
      setAssets([]);
      return;
    }
    setLoading(true);
    try {
      const params = new URLSearchParams({
        p: '1',
        size: '50',
        group_id: String(activeGroupId),
      });
      if (keyword.trim()) params.set('keyword', keyword.trim());
      const res = await API.get(`/api/seedance/assets?${params.toString()}`);
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      setAssets(res.data.data.items || []);
    } finally {
      setLoading(false);
    }
  }, [activeGroupId, keyword]);

  useEffect(() => {
    loadGroups();
  }, [loadGroups]);

  useEffect(() => {
    loadAssets();
  }, [loadAssets]);

  const handleCreateGroup = async () => {
    const res = await API.post('/api/seedance/groups', {
      name: groupName,
      group_type: 'AIGC',
    });
    if (!res.data.success) {
      showError(res.data.message);
      return;
    }
    showSuccess(t('Asset group created'));
    setGroupModalOpen(false);
    setGroupName('');
    await loadGroups();
  };

  const handleCreateAsset = async () => {
    const res = await API.post('/api/seedance/assets', {
      group_id: activeGroupId,
      name: assetName,
      asset_type: assetType,
      url: assetUrl,
    });
    if (!res.data.success) {
      showError(res.data.message);
      return;
    }
    showSuccess(t('Asset created'));
    setAssetModalOpen(false);
    setAssetName('');
    setAssetUrl('');
    await loadAssets();
  };

  const handleDeleteAsset = async (record) => {
    Modal.confirm({
      title: t('Delete asset'),
      content: t('Delete this asset? This action cannot be undone.'),
      onOk: async () => {
        const res = await API.delete(`/api/seedance/assets/${record.id}`);
        if (!res.data.success) {
          showError(res.data.message);
          return;
        }
        showSuccess(t('Asset deleted'));
        await loadAssets();
      },
    });
  };

  const handleSyncAsset = async (record) => {
    const res = await API.get(`/api/seedance/assets/${record.id}?sync=true`);
    if (!res.data.success) {
      showError(res.data.message);
      return;
    }
    showSuccess(t('Asset synced from upstream'));
    await loadAssets();
  };

  const columns = useMemo(
    () => [
      { title: t('Name'), dataIndex: 'name' },
      { title: t('Asset type'), dataIndex: 'asset_type' },
      { title: t('Public URL'), dataIndex: 'public_url' },
      {
        title: t('Reference'),
        render: (_, record) => `seedance_asset://${record.id}`,
      },
      {
        title: t('Edit'),
        render: (_, record) => (
          <>
            <Button size='small' onClick={() => handleSyncAsset(record)}>
              {t('Sync from upstream')}
            </Button>
            <Button
              size='small'
              type='danger'
              style={{ marginLeft: 8 }}
              onClick={() => handleDeleteAsset(record)}
            >
              {t('Delete asset')}
            </Button>
          </>
        ),
      },
    ],
    [t],
  );

  return (
    <div className='mt-[60px] px-2'>
      <div className='mb-4 flex items-center justify-between gap-3'>
        <Title heading={4}>{t('Seedance Assets')}</Title>
        <div className='flex gap-2'>
          <Button onClick={() => setGroupModalOpen(true)}>
            {t('New asset group')}
          </Button>
          <Button
            theme='solid'
            disabled={!activeGroupId}
            onClick={() => setAssetModalOpen(true)}
          >
            {t('Add asset')}
          </Button>
        </div>
      </div>

      <Card className='mb-4'>
        <Title heading={6}>{t('How to use Seedance assets in video generation')}</Title>
        <Text type='secondary' className='block mb-2'>
          {t('Upload assets here and copy the seedance_asset:// reference.')}
        </Text>
        <Text type='secondary' className='block mb-2'>
          {t('Call POST /v1/video/generations with your API key (same account).')}
        </Text>
        <pre className='bg-gray-50 p-3 rounded text-xs overflow-x-auto'>
{`POST /v1/video/generations
{
  "model": "JDseedance2.0-10",
  "prompt": "...",
  "duration": 11,
  "metadata": {
    "ratio": "16:9",
    "generate_audio": true,
    "watermark": false,
    "content": [{
      "type": "image_url",
      "image_url": { "url": "seedance_asset://1" },
      "role": "reference_image"
    }]
  }
}`}
        </pre>
      </Card>

      <div className='grid gap-4 md:grid-cols-[240px_1fr]'>
        <Card title={t('Asset groups')}>
          {groups.map((group) => (
            <div
              key={group.id}
              className='mb-2 flex cursor-pointer items-center justify-between rounded px-2 py-2 hover:bg-gray-50'
              onClick={() => setActiveGroupId(group.id)}
            >
              <Text strong={activeGroupId === group.id}>{group.name}</Text>
              {group.is_default ? <Tag size='small'>{t('Default')}</Tag> : null}
            </div>
          ))}
        </Card>

        <Card>
          <Input
            className='mb-4 max-w-sm'
            placeholder={t('Search assets by name')}
            value={keyword}
            onChange={setKeyword}
          />
          <Table
            columns={columns}
            dataSource={assets}
            loading={loading}
            pagination={false}
            rowKey='id'
          />
        </Card>
      </div>

      <Modal
        title={t('New asset group')}
        visible={groupModalOpen}
        onCancel={() => setGroupModalOpen(false)}
        onOk={handleCreateGroup}
        okButtonProps={{ disabled: !groupName.trim() }}
      >
        <Input
          placeholder={t('Name')}
          value={groupName}
          onChange={setGroupName}
        />
      </Modal>

      <Modal
        title={t('Add asset')}
        visible={assetModalOpen}
        onCancel={() => setAssetModalOpen(false)}
        onOk={handleCreateAsset}
        okButtonProps={{
          disabled: !assetType.trim() || !assetUrl.trim() || !activeGroupId,
        }}
      >
        <div className='space-y-3'>
          <Input
            placeholder={t('Name')}
            value={assetName}
            onChange={setAssetName}
          />
          <Select
            value={assetType}
            onChange={setAssetType}
            optionList={[
              { value: 'Image', label: 'Image' },
              { value: 'Video', label: 'Video' },
              { value: 'Audio', label: 'Audio' },
            ]}
            style={{ width: '100%' }}
          />
          <Input
            placeholder={t('Public URL')}
            value={assetUrl}
            onChange={setAssetUrl}
          />
        </div>
      </Modal>
    </div>
  );
};

export default SeedanceAssets;
