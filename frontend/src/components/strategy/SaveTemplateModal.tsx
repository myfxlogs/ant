import React from 'react';
import { Form, Input, Modal } from 'antd';
import { useTranslation } from 'react-i18next';

type Props = {
	open: boolean;
	confirmLoading: boolean;
	form: any;
	onCancel: () => void;
	onOk: () => void;
	afterOpenChange?: (open: boolean) => void;
};

const SaveTemplateModal: React.FC<Props> = ({ open, confirmLoading, form, onCancel, onOk, afterOpenChange }) => {
	const { t } = useTranslation();
	return (
		<Modal
			title={t('strategy.templateModal.title')}
			open={open}
			onCancel={onCancel}
			onOk={onOk}
			okText={t('common.save')}
			cancelText={t('common.cancel')}
			confirmLoading={confirmLoading}
			afterOpenChange={afterOpenChange}
			destroyOnHidden
		>
			<Form form={form} layout="vertical">
				<Form.Item name="name" label={t('strategy.templateModal.fields.name')} rules={[{ required: true }]}>
					<Input placeholder={t('strategy.templateModal.placeholders.name')} />
				</Form.Item>
				<Form.Item name="description" label={t('strategy.templateModal.fields.description')}>
					<Input placeholder={t('strategy.templateModal.placeholders.description')} />
				</Form.Item>
			</Form>
		</Modal>
	);
};

export default SaveTemplateModal;
