import React from 'react';
import { Button, Content, TextInput } from '@patternfly/react-core';
import { Modal, ModalVariant } from '@patternfly/react-core/deprecated';
import { ActionOperations } from '@app/types/types';

interface ClusterActionConfirmProps {
  isOpen: boolean;
  clusterId: string;
  actionOperation: ActionOperations | null;
  onConfirm: () => void;
  onClose: () => void;
}

export const ClusterActionConfirm: React.FunctionComponent<ClusterActionConfirmProps> = ({
  isOpen,
  clusterId,
  actionOperation,
  onConfirm,
  onClose,
}) => {
  const [confirmText, setConfirmText] = React.useState('');

  React.useEffect(() => {
    if (!isOpen) {
      setConfirmText('');
    }
  }, [isOpen]);

  if (!isOpen || !actionOperation) {
    return null;
  }

  const isDelete = actionOperation === ActionOperations.DELETE_CLUSTER;
  const isDeleteConfirmed = confirmText === clusterId;

  return (
    <Modal
      variant={ModalVariant.small}
      title={isDelete ? 'Confirm cluster deletion' : 'Confirm power action'}
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button
          key="confirm"
          variant={isDelete ? 'danger' : 'primary'}
          onClick={onConfirm}
          isDisabled={isDelete && !isDeleteConfirmed}
        >
          {isDelete ? 'Delete' : 'Confirm'}
        </Button>,
        <Button key="cancel" variant="link" onClick={onClose}>
          Cancel
        </Button>,
      ]}
    >
      <Content component="p">
        {isDelete ? (
          <>
            Are you sure you want to <strong>delete</strong> the cluster <strong>{clusterId}</strong>? This action is{' '}
            <strong>irreversible</strong> and will destroy all cluster resources.
          </>
        ) : (
          <>
            Are you sure you want to <strong>{actionOperation}</strong> the cluster <strong>{clusterId}</strong>?
          </>
        )}
      </Content>
      {isDelete && (
        <Content component="p" className="pf-v6-u-mt-md">
          Type <strong>{clusterId}</strong> to confirm:
          <TextInput
            className="pf-v6-u-mt-sm"
            value={confirmText}
            onChange={(_event, value) => setConfirmText(value)}
            aria-label="Type cluster ID to confirm deletion"
            placeholder={clusterId}
          />
        </Content>
      )}
    </Modal>
  );
};
